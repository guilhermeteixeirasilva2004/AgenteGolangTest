package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// --- FERRAMENTAS DO SISTEMA (TOOLS) ---

func ferramentaDataAtual() string {
	agora := time.Now()
	return agora.Format("02/01/2006 às 15:04")
}

func ferramentaContarLetras(texto string) string {
	textoLimpo := strings.ReplaceAll(texto, " ", "")
	tamanho := len([]rune(textoLimpo))
	return fmt.Sprintf("A palavra '%s' tem %d letras.", texto, tamanho)
}

// --------------------------------------

func main() {
	fmt.Println("=======================================")
	fmt.Println("🧠 Agente ReAct + Streaming ⚡")
	fmt.Println("🛠️  Ferramentas: [data_atual, contar_letras]")
	fmt.Println("=======================================")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nVocê: ")
		entrada, _ := reader.ReadString('\n')
		entrada = strings.TrimSpace(entrada)

		if strings.ToLower(entrada) == "sair" {
			break
		}
		if entrada == "" {
			continue
		}

		// NOVO PROMPT ESTRITO: Regras militares para o modelo não se perder
		historicoContexto := `<|im_start|>system
Você é um agente inteligente. Você deve pensar passo a passo.
REGRA 1: Você só pode usar UMA ferramenta por vez. Se precisar de duas informações, use uma ferramenta, espere o resultado, e depois use a outra.
REGRA 2: Você NÃO SABE a data de hoje. SEMPRE use a ferramenta data_atual quando perguntarem sobre datas ou dias.

Ferramentas disponíveis:
- data_atual: Retorna a data e hora. Entrada deve ser vazia.
- contar_letras: Conta a quantidade de letras. Entrada deve ser a palavra.

Use EXATAMENTE este formato:
Pensamento: [o que fazer agora]
Ação: [data_atual, contar_letras ou nenhuma]
Entrada: [valor da entrada ou vazio]

Quando tiver todas as informações, use:
Pensamento: Já tenho tudo.
Resposta Final: [sua resposta]<|im_end|>
<|im_start|>user
` + entrada + "<|im_end|>\n<|im_start|>assistant\n"

		// O Loop ReAct (Máximo de 5 passos)
		for passo := 1; passo <= 5; passo++ {

			// 1. Pede para a IA pensar e agir com Streaming
			respostaLLM := chamarLLMStream(historicoContexto)

			historicoContexto += respostaLLM

			// Verifica se já encerrou
			if strings.Contains(respostaLLM, "Resposta Final:") {
				break
			}

			// Verifica se pediu ferramenta
			if strings.Contains(respostaLLM, "Ação:") {
				acao := extrairLinha(respostaLLM, "Ação:")
				entradaFerramenta := extrairLinha(respostaLLM, "Entrada:")

				fmt.Printf("\n\n⚙️  [SISTEMA EXECUTANDO FERRAMENTA: %s('%s')]\n", acao, entradaFerramenta)

				var observacao string
				switch strings.ToLower(acao) {
				case "data_atual":
					observacao = ferramentaDataAtual()
				case "contar_letras":
					observacao = ferramentaContarLetras(entradaFerramenta)
				default:
					observacao = "Ferramenta não encontrada."
				}

				fmt.Println("🔍 Observação:", observacao, "\n")

				// Devolve a observação pro histórico
				historicoContexto += "\nObservação: " + observacao + "\n"
			} else {
				break
			}
		}
	}
}

// Lemos a stream, imprimimos na tela e salvamos a string completa
func chamarLLMStream(prompt string) string {
	dados := map[string]interface{}{
		"prompt":      prompt,
		"n_predict":   300,
		"temperature": 0.1,
		"stop":        []string{"<|im_end|>", "Observação:"},
		"stream":      true,
	}

	jsonDados, _ := json.Marshal(dados)
	resp, err := http.Post("http://127.0.0.1:8085/completion", "application/json", bytes.NewBuffer(jsonDados))
	if err != nil {
		return "Erro de conexão."
	}
	defer resp.Body.Close()

	var textoCompleto string
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		linha := scanner.Text()
		if strings.HasPrefix(linha, "data: ") {
			jsonPuro := strings.TrimPrefix(linha, "data: ")

			var pedaco map[string]interface{}
			json.Unmarshal([]byte(jsonPuro), &pedaco)

			if conteudo, ok := pedaco["content"].(string); ok {
				fmt.Print(conteudo)
				textoCompleto += conteudo
			}
		}
	}
	return textoCompleto
}

// Extrai o valor de uma linha (ex: "Ação: data_atual" -> "data_atual")
func extrairLinha(textoCompleto string, chave string) string {
	linhas := strings.Split(textoCompleto, "\n")
	for _, linha := range linhas {
		if strings.HasPrefix(strings.TrimSpace(linha), chave) {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(linha), chave))
		}
	}
	return ""
}
