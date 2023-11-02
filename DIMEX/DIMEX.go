// ALUNOS:
// - Eduardo Garcia
// - Eduardo Riboli
// - Jocemar Nicolodi
// - Matheus Fernandes

/*  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
    Professor: Fernando Dotti  (https://fldotti.github.io/)
    Modulo representando Algoritmo de Exclusão Mútua Distribuída:
    Semestre 2023/1
	Aspectos a observar:
	   mapeamento de módulo para estrutura
	   inicializacao
	   semantica de concorrência: cada evento é atômico
	   							  módulo trata 1 por vez
	Q U E S T A O
	   Além de obviamente entender a estrutura ...
	   Implementar o núcleo do algoritmo ja descrito, ou seja, o corpo das
	   funcoes reativas a cada entrada possível:
	   			handleUponReqEntry()  // recebe do nivel de cima (app)
				handleUponReqExit()   // recebe do nivel de cima (app)
				handleUponDeliverRespOk(msgOutro)   // recebe do nivel de baixo
				handleUponDeliverReqEntry(msgOutro) // recebe do nivel de baixo
*/

package DIMEX

import (
	PP2PLink "SD/PP2PLink"
	"fmt"
	"strconv"
	"strings"
)

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type State int // enumeracao dos estados possiveis de um processo
const (
	noMX State = iota
	wantMX
	inMX
)

type dmxReq int // enumeracao dos estados possiveis de um processo
const (
	ENTER dmxReq = iota
	EXIT
)

type dmxResp struct { // mensagem do módulo DIMEX infrmando que pode acessar - pode ser somente um sinal (vazio)
	// mensagem para aplicacao indicando que pode prosseguir
}

type DIMEX_Module struct {
	Req       chan dmxReq  // canal para receber pedidos da aplicacao (REQ e EXIT)
	Ind       chan dmxResp // canal para informar aplicacao que pode acessar
	addresses []string     // endereco de todos, na mesma ordem
	id        int          // identificador do processo - é o indice no array de enderecos acima
	st        State        // estado deste processo na exclusao mutua distribuida
	waiting   []bool       // processos aguardando tem flag true
	lcl       int          // relogio logico local
	reqTs     int          // timestamp local da ultima requisicao deste processo
	nbrResps  int
	dbg       bool

	Pp2plink *PP2PLink.PP2PLink // acesso aa comunicacao enviar por PP2PLinq.Req  e receber por PP2PLinq.Ind
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDIMEX(_addresses []string, _id int, _dbg bool) *DIMEX_Module {

	p2p := PP2PLink.NewPP2PLink(_addresses[_id], _dbg)

	dmx := &DIMEX_Module{
		Req: make(chan dmxReq),
		Ind: make(chan dmxResp),

		addresses: _addresses,
		id:        _id,
		st:        noMX,
		waiting:   make([]bool, len(_addresses)),
		lcl:       0,
		reqTs:     0,
		dbg:       _dbg,

		Pp2plink: p2p}

	dmx.Start()
	dmx.outDbg("Init DIMEX!")
	return dmx
}

// ------------------------------------------------------------------------------------
// ------- nucleo do funcionamento
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) Start() {

	go func() {
		for {
			select {
			// Request vind da aplicação
			case dmxR := <-module.Req:
				// Processo quer entrar
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry()

				} else if dmxR == EXIT {
					// Processo quer sair
					module.outDbg("app libera mx")
					module.handleUponReqExit()
				}

			// Request vindo de outro processo
			case msgOutro := <-module.Pp2plink.Ind:

				// Se a mensagem que outro processo envou for de OK
				if strings.Contains(msgOutro.Message, "respOK") {

					// Eu contabilizo essa resposta e vejo se todos os processos também
					// deram OK, se todos deram, eu posso entrar na SC
					module.outDbg("         <<<---- responde! " + msgOutro.Message)
					module.handleUponDeliverRespOk(msgOutro)

				} else if strings.Contains(msgOutro.Message, "reqEntry") {
					// Se a mensagem que outro processo enviou for solicitando entrada
					// na SC, eu respondo OK para ele
					module.outDbg("          <<<---- pede??  " + msgOutro.Message)
					module.handleUponDeliverReqEntry(msgOutro)

				}
			}
		}
	}()
}

// ------------------------------------------------------------------------------------
// ------- tratamento de pedidos vindos da aplicacao
// ------- UPON ENTRY
// ------- UPON EXIT
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponReqEntry() {
	/*
					upon event [ dmx, Entry  |  r ]  do
		    			lts.ts++
		    			myTs := lts
		    			resps := 0
		    			para todo processo p
							trigger [ pl , Send | [ reqEntry, r, myTs ]
		    			estado := queroSC
	*/
	module.lcl++              // atualiza o relogio local (lts.ts++)
	module.reqTs = module.lcl // atualiza o timestamp local (myTs := lts)
	module.nbrResps = 0       // zera numero de respostas (resps := 0)

	// Envia para todos os outros processos uma mensagem falando q eu quero entrar
	for i := 0; i < len(module.addresses); i++ {
		// IF para ñ mandar para eu mesmo
		if module.id != i {

			message := fmt.Sprintf("||reqEntry||%d||%d||", module.id, module.reqTs)
			module.sendToLink(module.addresses[i], message)
		}
	}
	// Fala que o processo quer acessar
	module.st = wantMX
}

func (module *DIMEX_Module) handleUponReqExit() {
	/*
						upon event [ dmx, Exit  |  r  ]  do
		       				para todo [p, r, ts ] em postergados
		          				trigger [ pl, Send | p , [ respOk, r ]  ]
		    				estado := naoQueroSC
	*/

	// Para todos os processos que estão esperando, eu falo que eu não quero mais acessar
	// for i := 0; i < len(module.addresses); i++ {
	// 	// IF para ñ mandar para eu mesmo
	// 	if module.id != i {
	// 		module.sendToLink(module.addresses[i], "||respOK")
	// 	}
	// }

	// Fala que o processo nao quer mais acessar
	module.st = noMX
}

// ------------------------------------------------------------------------------------
// ------- tratamento de mensagens de outros processos
// ------- UPON respOK
// ------- UPON reqEntry
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponDeliverRespOk(msgOutro PP2PLink.PP2PLink_Ind_Message) { // deve estar completo conforme algoritmo.
	/*
						upon event [ pl, Deliver | p, [ respOk, r ] ]
		      				resps++
		      				se resps = N
		    				então trigger [ dmx, Deliver | free2Access ]
		  					    estado := estouNaSC

	*/

	// incremento falando pois um processo respondeu
	module.nbrResps++

	// Se todos responderam, libera a aplicacao para entrar na SC
	if module.nbrResps == (len(module.addresses) - 1) {
		module.outDbg(" resposta de todos - libera app para entrar mx")
		module.Ind <- dmxResp{}
	}
	// muda o estado falando q esto na SC
	module.st = inMX
}

func (module *DIMEX_Module) handleUponDeliverReqEntry(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	// outro processo quer entrar na SC
	/*
						upon event [ pl, Deliver | p, [ reqEntry, r, rts ]  do
		     				se (estado == naoQueroSC)   OR
		        				 (estado == QueroSC AND  myTs >  ts)
							então  trigger [ pl, Send | p , [ respOk, r ]  ]
		 					senão
		        				se (estado == estouNaSC) OR
		           					 (estado == QueroSC AND  myTs < ts)
		        				então  postergados := postergados + [p, r ]
		     					lts.ts := max(lts.ts, rts.ts)
	*/
	msgTerms := strings.Split(msgOutro.Message, "||")
	idOutro, _ := strconv.Atoi(msgTerms[2])
	module.sendToLink(module.addresses[idOutro], "||respOK")
}

// ------------------------------------------------------------------------------------
// ------- funcoes de ajuda
// ------------------------------------------------------------------------------------

// func (module *DIMEX_Module) sendToLink(address string, content string, space string) {
// 	module.outDbg(space + " ---->>>>   to: " + address + "     msg: " + content)
// 	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
// 		To:      address,
// 		Message: content}
// }

func (module *DIMEX_Module) sendToLink(address string, content string) {
	module.outDbg(" envia no pplink: " + address + "    " + content)
	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
		To:      address,
		Message: content}
}

func before(oneId, oneTs, othId, othTs int) bool {
	if oneTs < othTs {
		return true
	} else if oneTs > othTs {
		return false
	} else {
		return oneId < othId
	}
}

func (module *DIMEX_Module) outDbg(s string) {
	if module.dbg {
		fmt.Println(". . . . . . . . . . . . [ DIMEX : " + s + " ]")
	}
}
