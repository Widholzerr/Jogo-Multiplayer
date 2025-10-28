package main

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"sync"
)

type Servidor struct {
	mu          sync.Mutex
	jogadores   map[string]*Jogador
	processados map[string]int
}

func NovoServidor() *Servidor {
	return &Servidor{
		jogadores:   make(map[string]*Jogador),
		processados: make(map[string]int),
	}
}

func (s *Servidor) RegistrarJogador(id string, reply *Jogador) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, existe := s.jogadores[id]; existe {
		return errors.New("jogador já registrado")
	}

	s.jogadores[id] = &Jogador{ID: id, PosX: 2, PosY: 2, Vidas: 3}
	*reply = *s.jogadores[id]
	fmt.Println("[Servidor] Novo jogador:", id)
	return nil
}

func (s *Servidor) AtualizarPosicao(j *Jogador, reply *bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if last, ok := s.processados[j.ID]; ok && j.SeqNumber <= last {
		fmt.Println("[Servidor] Ignorando repetição:", j.ID, j.SeqNumber)
		*reply = true
		return nil
	}
	s.processados[j.ID] = j.SeqNumber

	if jog, ok := s.jogadores[j.ID]; ok {
		jog.PosX, jog.PosY = j.PosX, j.PosY
		fmt.Println("[Servidor] Atualizado:", jog.ID, jog.PosX, jog.PosY)
		*reply = true
		return nil
	}
	return errors.New("jogador não encontrado")
}

func (s *Servidor) ObterEstado(_ int, estado *EstadoJogo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copia := make(map[string]Jogador)
	for id, jog := range s.jogadores {
		copia[id] = *jog
	}
	estado.Jogadores = copia
	return nil
}

func main() {
	srv := NovoServidor()
	rpc.Register(srv)

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	fmt.Println("[Servidor] Esperando conexões na porta 8080...")

	for {
		conn, _ := ln.Accept()
		go rpc.ServeConn(conn)
	}
}
