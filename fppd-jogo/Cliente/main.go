package main

import (
	"fmt"
	"math/rand"
	"net/rpc"
	"os"
	"time"
)

func main() {

	var rpcClient *rpc.Client
	var meuID string
	var seq int
	var err error

	//Conexão com servidor
	rpcClient, err = rpc.Dial("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}

	meuID = fmt.Sprintf("Jogador-%d", rand.Intn(10000))
	var jogador Jogador
	if err := rpcClient.Call("Servidor.RegistrarJogador", meuID, &jogador); err != nil {
		panic(err)
	}
	fmt.Println("[Cliente] Conectado ao servidor como", meuID)

	// Carregamento da interface e mapa
	interfaceIniciar()
	defer interfaceFinalizar()

	mapaFile := "mapa.txt"
	if len(os.Args) > 1 {
		mapaFile = os.Args[1]
	}

	jogo := jogoNovo()
	if err := jogoCarregarMapa(mapaFile, &jogo); err != nil {
		panic(err)
	}

	interfaceDesenharJogo(&jogo)

	// goroutine para atualizar estado dos outros jogadores
	go func() {
		for {
			var estado EstadoJogo
			err := rpcClient.Call("Servidor.ObterEstado", 0, &estado)
			if err == nil {
				lock()

				// Limpa o mapa, mantendo paredes, vegetação e armadilhas
				for y := range jogo.Mapa {
					for x := range jogo.Mapa[y] {
						sim := jogo.Mapa[y][x].simbolo
						if sim == '☺' || sim == '●' {
							jogo.Mapa[y][x] = Vazio
						}
					}
				}

				// Redesenha todos os jogadores conectados
				for _, jog := range estado.Jogadores {
					if jog.ID == meuID {
						// Atualiza posição local (mantém personagem local no mapa)
						jogo.PosX, jogo.PosY = jog.PosX, jog.PosY
						jogo.Mapa[jog.PosY][jog.PosX] = Elemento{'☺', CorCinzaEscuro, CorPadrao, true}
					} else {
						// Desenha outros jogadores (● azul)
						jogo.Mapa[jog.PosY][jog.PosX] = Elemento{'●', CorCiano, CorPadrao, true}
					}
				}

				unlock()
				interfaceDesenharJogo(&jogo)
			}

			time.Sleep(300 * time.Millisecond)
		}
	}()

	// Canal da armadilha
	canalArmadilha := make(chan string, 5)

	// Atualiza status e redesenha as armadilhas do jogo
	go func() {
		for msg := range canalArmadilha {
			lock()
			jogo.StatusMsg = msg
			interfaceDesenharJogo(&jogo)
			unlock()
		}
	}()

	// ELEMENTOS CONCORRENTES
	iniciarInimigo(&jogo, 10, 4)
	iniciarArmadilha(&jogo, 25, 5, canalArmadilha)
	iniciarNPCComComunicacao(&jogo, 15, 10)

	// movimento local + envio RPC
	for {
		evento := interfaceLerEventoTeclado()
		if continuar := personagemExecutarAcao(evento, &jogo); !continuar {
			break
		}
		interfaceDesenharJogo(&jogo)

		// Envia posição atual ao servidor (exactly-once via SeqNumber)
		seq++
		j := Jogador{ID: meuID, PosX: jogo.PosX, PosY: jogo.PosY, SeqNumber: seq}
		var ok bool
		err = rpcClient.Call("Servidor.AtualizarPosicao", &j, &ok)
		if err != nil {
			fmt.Println("[Cliente] Falha RPC, reenviando...")
			_ = rpcClient.Call("Servidor.AtualizarPosicao", &j, &ok)
		}
	}
}
