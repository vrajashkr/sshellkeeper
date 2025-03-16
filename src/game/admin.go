package game

type GameAdmin interface {
	GetPlayerRecord(username string) (Player, error)
	CheckPlayerAnswer(player Player, answer string) error
	PromotePlayer(player Player) error
}
