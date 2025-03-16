package game

import (
	"fmt"
	"log/slog"

	"github.com/vrajashkr/sshellkeeper/src/sshserver"
	"golang.org/x/crypto/ssh"
)

type GameEngine struct {
	gadmin GameAdmin
}

func NewGameEngine(gAdmin GameAdmin) GameEngine {
	return GameEngine{
		gadmin: gAdmin,
	}
}

func handleChannelWriteFailure(logger slog.Logger, err error) {
	if err != nil {
		logger.Error("failed to write message to channel", "error", err.Error())
	}
}

func (ge *GameEngine) RunPlayerGameFlow(username string, ch ssh.Channel) {
	logger := slog.Default().With("username", username)

	lines := []string{
		fmt.Sprintf("Welcome %s to the Keeper of the Game", username),
		"You are now accessing a highly restricted resource.",
		"Your usage is monitored.",
		"Unauthorized use is forbidden.",
	}
	err := sshserver.WriteLinesToChan(ch, lines)
	handleChannelWriteFailure(*logger, err)

	player, err := ge.gadmin.GetPlayerRecord(username)
	if err != nil {
		logger.Error("failed to get player record", "error", err.Error())
		err = sshserver.WriteLinesToChan(ch, []string{"Uh oh! Something went wrong! Try again later."})
		handleChannelWriteFailure(*logger, err)
		return
	}

	if player.CurrentQuestionNumber >= len(player.Questions) {
		logger.Info("user logged in after game completion")
		err := sshserver.WriteLinesToChan(ch, []string{"Congration! Ya done it! Don't log in again. Bye."})
		handleChannelWriteFailure(*logger, err)
		return
	}

	currentQuestion := player.Questions[player.CurrentQuestionNumber]

	lines = []string{
		fmt.Sprintf("You are currently on question number %d.", player.CurrentQuestionNumber),
		"",
		"Question",
		"========================",
		currentQuestion.Question,
		"========================",
		"",
		"What is your answer?",
	}

	err = sshserver.WriteLinesToChan(ch, lines)
	handleChannelWriteFailure(*logger, err)

	answer, err := sshserver.ReadDataFromChannel(*logger, ch)
	if err != nil {
		logger.Error("failed to read answer from channel", "error", err.Error())
		err = sshserver.WriteLinesToChan(ch, []string{"Uh oh! Something went wrong! Try again later."})
		handleChannelWriteFailure(*logger, err)
		return
	}

	logger.Debug("read answer from channel", "answer", answer)

	err = ge.gadmin.CheckPlayerAnswer(player, answer)
	if err != nil {
		logger.Error("failed to bind to player credentials", "error", err.Error())
		err = sshserver.WriteLinesToChan(ch, []string{"That wasn't correct!", "Try again! Bye for now uWu!"})
		handleChannelWriteFailure(*logger, err)
		return
	}

	err = sshserver.WriteLinesToChan(ch, []string{"", "Ya got that right! Not bad."})
	handleChannelWriteFailure(*logger, err)

	err = ge.gadmin.PromotePlayer(player)
	if err != nil {
		logger.Error("failed to promote player", "error", err.Error())
		err = sshserver.WriteLinesToChan(ch, []string{"Uh oh! Something went wrong! Try again later."})
		handleChannelWriteFailure(*logger, err)
		return
	}
	err = sshserver.WriteLinesToChan(ch, []string{"Login again for your next question."})
	handleChannelWriteFailure(*logger, err)

	logger.Info("finished player flow. Terminating session")
}
