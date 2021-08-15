# Trabalho de Raft

## Tarefa
Implementar eleição de líder e heartbeats (RPCs AppendEntries sem
entradas no log) conforme a especificação do Raft. O objetivo é que um
único líder seja eleito, que o líder continue sendo o líder se não houver
falhas e que um novo líder assuma o controle se o antigo líder falhar ou
se os pacotes de/para o antigo líder forem perdidos.

## Dicas

* Adicione qualquer estado necessário à struct Raft em raft.go. Você também precisará definir uma estrutura para armazenar informações sobre cada entrada de log. Seu código deve seguir a Figura 2 (página 4) da versão estendida do [artigo sobre Raft](https://www.cs.bu.edu/~jappavoo/jappavoo.github.com/451/papers/raft-extended.pdf) o mais próximo possível.

* Preencha as estruturas RequestVoteArgs e RequestVoteReply. Modifique Make () para criar uma rotina go em segundo plano que iniciará a eleição do líder periodicamente enviando RPCs de RequestVote quando um participante não receber notícias de outro participante por um tempo. Dessa maneira, um participante aprenderá quem é o líder, se ele já existe, ou se tornará o líder. Implemente o método RPC RequestVote () para que os servidores votem um no outro.

* Para implementar heartbeats, defina uma struct RPC AppendEntries (embora você não precise de todos os seus argumentos) e peça ao líder para enviá-los periodicamente. Escreva um método da RPC AppendEntries que redefina o tempo limite da eleição para que outros servidores não ajam como líderes quando um já tiver sido eleito.

* Certifique-se de que o tempo limite das eleições em diferentes participantes nem sempre seja acionado ao mesmo tempo, ou então todos os participantes votarão em si mesmos e ninguém se tornará líder.