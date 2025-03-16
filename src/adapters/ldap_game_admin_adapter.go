package adapters

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-ldap/ldap/v3"
	"github.com/spf13/viper"
	"github.com/vrajashkr/sshellkeeper/src/game"
	"github.com/vrajashkr/sshellkeeper/src/ldaptools"
)

type LdapGameAdmin struct {
	ldapConn *ldaptools.LdapAdminConnection
}

func NewLdapGameAdmin(ldapConn *ldaptools.LdapAdminConnection) LdapGameAdmin {
	return LdapGameAdmin{
		ldapConn: ldapConn,
	}
}

func (lga *LdapGameAdmin) GetPlayerRecord(username string) (game.Player, error) {
	userEntry, err := lga.ldapConn.GetUserRecord(username, []string{"dn", "question_text", "memberof", "uid"})
	if err != nil {
		return game.Player{}, err
	}

	p, err := newPlayerFromLdapEntry(userEntry)
	if err != nil {
		return game.Player{}, err
	}

	return p, nil
}

func (lga *LdapGameAdmin) CheckPlayerAnswer(player game.Player, answer string) error {
	return lga.ldapConn.TryBindLdap(player.DistinguishedName, answer)
}

func (lga *LdapGameAdmin) PromotePlayer(player game.Player) error {
	newPassword := viper.GetString("game.completion_password")
	nextLevel := player.CurrentQuestionNumber + 1

	if nextLevel < len(player.Questions) {
		// there are some more questions available
		newPassword = player.Questions[nextLevel].Answer
	}

	err := lga.ldapConn.SetUserPassword(player.DistinguishedName, newPassword)
	if err != nil {
		return err
	}

	err = lga.ldapConn.AddUserToGroup(player.Username, fmt.Sprintf("%d", nextLevel))
	if err != nil {
		return err
	}

	return nil
}

func newPlayerFromLdapEntry(userRecord *ldap.Entry) (game.Player, error) {
	playerQuestions, err := questionsFromLdapEntry(userRecord)
	if err != nil {
		return game.Player{}, err
	}
	groups := userRecord.GetAttributeValues("memberof")
	currentQNo, err := currentQuestionNumber(groups)
	if err != nil {
		return game.Player{}, err
	}

	return game.Player{
		Username:              userRecord.GetAttributeValue("uid"),
		DistinguishedName:     userRecord.DN,
		Questions:             playerQuestions,
		Groups:                groups,
		CurrentQuestionNumber: currentQNo,
	}, nil
}

func currentQuestionNumber(groups []string) (int, error) {
	minGroup := 0

	for _, grp := range groups {
		level, err := strconv.Atoi(ldaptools.ExtractCNfromDN(grp))
		if err != nil {
			return -1, err
		}
		if level > minGroup {
			minGroup = level
		}
	}

	return minGroup, nil
}

func questionsFromLdapEntry(user *ldap.Entry) ([]game.Question, error) {
	rawQuestions := user.GetAttributeValues("question_text")
	questions := make([]game.Question, len(rawQuestions))

	for idx, rawQ := range rawQuestions {
		q := game.Question{}
		err := json.Unmarshal([]byte(rawQ), &q)
		if err != nil {
			return []game.Question{}, err
		}
		questions[idx] = q
	}

	return questions, nil
}
