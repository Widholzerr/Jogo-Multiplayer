package main

type Jogador struct {
	ID         string
	PosX, PosY int
	Vidas      int
	SeqNumber  int
}

type EstadoJogo struct {
	Jogadores map[string]Jogador
}
