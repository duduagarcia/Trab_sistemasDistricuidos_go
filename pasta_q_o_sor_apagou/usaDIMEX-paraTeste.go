// Construido como parte da disciplina: Sistemas Distribuidos - PUCRS - Escola Politecnica
//  Professor: Fernando Dotti  (https://fldotti.github.io/)
/* fmt.Println("go run usaDIMEX.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
   fmt.Println("go run usaDIMEX.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
   fmt.Println("go run usaDIMEX.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
/*
  LANCAR N PROCESSOS EM SHELL's DIFERENTES, UMA PARA CADA PROCESSO.
  para cada processo: seu id único e a mesma lista de processos.
  o endereco de cada processo é o dado na lista, na posicao do seu id.
  no exemplo acima o processo com id=1  usa a porta 6001 para receber e as portas
  5000 e 7002 para mandar mensagens respectivamente para processos com id=0 e 2
*/

package main

import (
	"SD/DIMEX"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Please specify at least one address:port!")
		fmt.Println("go run usaDIMEX.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run usaDIMEX.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run usaDIMEX.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		return
	}

	id, _ := strconv.Atoi(os.Args[1])
	addresses := os.Args[2:]
	// fmt.Print("id: ", id, "   ") fmt.Println(addresses)

	var dmx *DIMEX.DIMEX_Module = DIMEX.NewDIMEX(addresses, id, true)
	fmt.Println(dmx)

	file, err := os.OpenFile("./mxOUT.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed at the end of the function

	time.Sleep(3 * time.Second)

	for {
		fmt.Println("[ APP id: ", id, " PEDE   MX ]")
		dmx.Req <- DIMEX.ENTER
		//fmt.Println("[ APP id: ", id, " ESPERA MX ]")
		<-dmx.Ind //

		_, err = file.WriteString("|") // marca entrada no arquivo
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		fmt.Println("[ APP id: ", id, " *EM*   MX ]")

		_, err = file.WriteString(".") // marca saida no arquivo
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		dmx.Req <- DIMEX.EXIT //
		fmt.Println("[ APP id: ", id, " FORA   MX ]")
	}
}
