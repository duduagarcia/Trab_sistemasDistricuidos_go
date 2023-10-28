/*  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
    Professor: Fernando Dotti  (https://fldotti.github.io/)
    Modulo representando Algoritmo de Exclusão Mútua Distribuída:
    Semestre 2023/1                                                      */

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
			case dmxR := <-module.Req: // vindo da  aplicação
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry() // ENTRADA DO ALGORITMO

				} else if dmxR == EXIT {
					module.outDbg("app libera mx")
					module.handleUponReqExit() // ENTRADA DO ALGORITMO
				}

			case msgOutro := <-module.Pp2plink.Ind: // vindo de outro processo
				//fmt.Printf("dimex recebe da rede: ", msgOutro)
				if strings.Contains(msgOutro.Message, "respOK") {
					module.outDbg("outro proc responde " + msgOutro.Message)
					module.handleUponDeliverRespOk(msgOutro) // ENTRADA DO ALGORITMO

				} else if strings.Contains(msgOutro.Message, "reqEntry") {
					module.outDbg("outro proc pede " + msgOutro.Message)
					module.handleUponDeliverReqEntry(msgOutro) // ENTRADA DO ALGORITMO

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

func (module *DIMEX_Module) handleUponReqEntry() { // segue algoritmo ... praticamente linha a linha
	/*
		upon event [ dmx, Entry  |  r ]  do
		    lts.ts++
		    myTs := lts
		    resps := 0
		    para todo processo p
				trigger [ pl , Send | [ reqEntry, r, myTs ]
		    estado := queroSC
	*/
	module.lcl++                                 // proximo evento
	module.reqTs = module.lcl                    // envio de request, pega o relogio local
	module.nbrResps = 0                          // zera numero de respostas
	for i := 0; i < len(module.addresses); i++ { // envio do pedido para cada outro processo
		if i != module.id { // nao manda para mim mesmo
			module.sendToLink(module.addresses[i], //  aqui define-se um protocolo:
				("||reqEntry||" + strconv.Itoa(module.id) + //       [ reqEntry||id||timestamp ]
					"||" + strconv.Itoa(module.reqTs)))
		}
	}
	module.st = wantMX // muda estado para quer acessar
}

func (module *DIMEX_Module) handleUponReqExit() {
	/*
							upon event [ dmx, Exit  |  r  ]  do
		       					para todo [p, r, ts ] em postergados           // para cada um que esta em waiting
		              				trigger [ pl, Send | p , [ respOk, r ]  ]  //    	module.sendToLink(address ..., "respOK")
		       					estado := naoQueroSC
	*/
	module.st = noMX // muda estado para nao quer quer acessar
}

func (module *DIMEX_Module) sendToLink(address string, content string) {
	module.outDbg(" envia no pplink: " + address + "    " + content)
	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
		To:      address,
		Message: content}
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
	module.nbrResps++
	if module.nbrResps == (len(module.addresses) - 1) {
		module.outDbg(" resposta de todos - libera app para entrar mx")
		module.Ind <- dmxResp{} // somente um sinal
	}
	module.st = inMX // muda estado para esta usando
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
	// NO MOMENTO RESPONDENDO QUE QUALQUER UM PODE ACESSAR
	msgTerms := strings.Split(msgOutro.Message, "||")
	idOutro, _ := strconv.Atoi(msgTerms[2]) // obtem identificador do outro processo, que vem na mensagem
	// tsOutro, _ := strconv.Atoi(msgTerms[3]) // obtem timestamo do outro processo, que vem na mensagem
	// fmt.Println("id ", idOutro, "   ts ", tsOutro)
	module.sendToLink(module.addresses[idOutro], "||respOK")
	// para deixar esperando colocar em waiting
	// no momento todos deixando entrar
}

// ------------------------------------------------------------------------------------
// ------- funcoes de ajuda
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) outDbg(s string) {
	if module.dbg {
		fmt.Println(". . . . . . . . . . . . [ DIMEX : " + s + " ]")
	}
}
