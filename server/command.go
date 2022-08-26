package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

const commandHelp = `* |/trivia create [quiz-name] [channel-name] | - Creates a Trivia Quiz for a specific channel.
* |/trivia list_quizes| - Lists the quizes that you have created
* |/trivia list_questions| - Lists the questions and answers that you have created for a specific quiz
* |/trivia add_question [quiz-name] [question * answer]| - Add a question and the answer to a a specific quiz.
* |/trivia delete_quiz [quiz-name]| - Deletes a specific quiz or all your quizes if you type *all* as quizname
* |/trivia delete_question [quiz-name] [question * answer] | Deletes a specific question 
* |/trivia start [quizname] | Starts a specific quiz 

`

const (
	commandTriggerCreate         = "create"
	commandTriggerListQuizes     = "list_quizes"
	commandTriggerListQuestions  = "list_questions"
	commandTriggerAddQuestion    = "add_question"
	commandTriggerDeleteQuiz     = "delete_quiz"
	commandTriggerDeleteQuestion = "delete_question"
	commandTriggerStart          = "start"
	welcomebotChannelWelcomeKey  = ""
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "trivia",
		DisplayName:      "Trivia Bot",
		Description:      "Trivia Bot lets you create and host a Trivia Quiz.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: create, add, next",
		AutoCompleteHint: "[command]",
		AutocompleteData: getAutocompleteData(),
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string, textArgs ...interface{}) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   fmt.Sprintf(text, textArgs...),
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) validateCommand(action string, parameters []string) string {
	switch action {
	case commandTriggerCreate:
		if len(parameters) != 2 {
			return "Please specify a channel and quiz name (one word, dashes allowed)."
		}
	case commandTriggerListQuizes:
		if len(parameters) > 0 {
			return "List command does not accept any extra parameters"
		}
	case commandTriggerListQuestions:
		if len(parameters) != 1 {
			return "`You need to provide the quiz name (one word, dashes allowed)"
		}
	case commandTriggerAddQuestion:
		if len(parameters) < 2 {
			return "`You need to provide the quiz name, the question and the answer. Split the answer and the question with a |"
		}
	case commandTriggerDeleteQuestion:
		if len(parameters) < 2 {
			return "You need to provide the quiz name and the question"
		}
	case commandTriggerStart:
		if len(parameters) != 1 {
			return "This function requires the quiz name"
		}
	}

	return ""
}

func (p *Plugin) executeCommandPreview(teamName string, args *model.CommandArgs) {
	found := false
	for _, message := range p.getWelcomeMessages() {
		var teamNamesArr = strings.Split(message.TeamName, ",")
		for _, name := range teamNamesArr {
			tn := strings.TrimSpace(name)
			if tn == teamName {
				p.postCommandResponse(args, "%s", teamName)
				if err := p.previewWelcomeMessage(teamName, args, *message); err != nil {
					p.postCommandResponse(args, "error occurred while processing greeting for team `%s`: `%s`", teamName, err)
					return
				}
				found = true
			}
		}
	}

	if !found {
		p.postCommandResponse(args, "team `%s` has not been found", teamName)
	}
}

func (p *Plugin) executeCommandList(args *model.CommandArgs) {
	wecomeMessages := p.getWelcomeMessages()

	if len(wecomeMessages) == 0 {
		p.postCommandResponse(args, "There are no welcome messages defined")
		return
	}

	// Deduplicate entries
	teams := make(map[string]struct{})
	for _, message := range wecomeMessages {
		teams[message.TeamName] = struct{}{}
	}

	var str strings.Builder
	str.WriteString("Teams for which welcome messages are defined:")
	for team := range teams {
		str.WriteString(fmt.Sprintf("\n * %s", team))
	}
	p.postCommandResponse(args, str.String())
}

func (p *Plugin) executeCommandSetWelcome(args *model.CommandArgs) {
	channelInfo, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		p.postCommandResponse(args, "error occurred while checking the type of the chanelId `%s`: `%s`", args.ChannelId, appErr)
		return
	}

	if channelInfo.Type == model.ChannelTypeDirect {
		p.postCommandResponse(args, "welcome messages are not supported for direct channels")
		return
	}

	// strings.Fields will consume ALL whitespace, so plain re-joining of the
	// parameters slice will not produce the same message
	message := strings.SplitN(args.Command, "set_channel_welcome", 2)[1]
	message = strings.TrimSpace(message)

	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	if appErr := p.API.KVSet(key, []byte(message)); appErr != nil {
		p.postCommandResponse(args, "error occurred while storing the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "stored the welcome message:\n%s", message)
}

func (p *Plugin) executeCommandGetWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the chanel: `%s`", appErr)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "welcome message has not been set yet")
		return
	}

	p.postCommandResponse(args, "Welcome message is:\n%s", string(data))
}

func (p *Plugin) executeCommandDeleteWelcome(args *model.CommandArgs) {
	key := fmt.Sprintf("%s%s", welcomebotChannelWelcomeKey, args.ChannelId)
	data, appErr := p.API.KVGet(key)

	if appErr != nil {
		p.postCommandResponse(args, "error occurred while retrieving the welcome message for the chanel: `%s`", appErr)
		return
	}

	if data == nil {
		p.postCommandResponse(args, "welcome message has not been set yet")
		return
	}

	if appErr := p.API.KVDelete(key); appErr != nil {
		p.postCommandResponse(args, "error occurred while deleting the welcome message for the chanel: `%s`", appErr)
		return
	}

	p.postCommandResponse(args, "welcome message has been deleted")
}

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	parameters := []string{}
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if command != "/trivia" {
		return &model.CommandResponse{}, nil
	}

	if response := p.validateCommand(action, parameters); response != "" {
		p.postCommandResponse(args, response)
		return &model.CommandResponse{}, nil
	}

	switch action {
	case commandTriggerCreate:
		teamName := parameters[0]
		p.executeCommandPreview(teamName, args)
		return &model.CommandResponse{}, nil
	case commandTriggerListQuizes:
		p.executeCommandList(args)
		return &model.CommandResponse{}, nil
	case commandTriggerListQuestions:
		p.executeCommandSetWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerAddQuestion:
		p.executeCommandGetWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerDeleteQuestion:
		p.executeCommandDeleteWelcome(args)
		return &model.CommandResponse{}, nil
	case commandTriggerStart:
		p.executeCommandDeleteWelcome(args)
		return &model.CommandResponse{}, nil
	case "":
		text := "###### Trivia Plugin - Slash Command Help\n" + strings.ReplaceAll(commandHelp, "|", "`")
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	p.postCommandResponse(args, "Unknown action %v", action)
	return &model.CommandResponse{}, nil
}

func getAutocompleteData() *model.AutocompleteData {
	welcomebot := model.NewAutocompleteData("welcomebot", "[command]",
		"Available commands: ")

	preview := model.NewAutocompleteData("preview", "[team-name]", "Preview the welcome message for the given team name")
	preview.AddTextArgument("Team name to preview welcome message", "[team-name]", "")
	welcomebot.AddCommand(preview)

	list := model.NewAutocompleteData("list", "", "Lists team welcome messages")
	welcomebot.AddCommand(list)

	setChannelWelcome := model.NewAutocompleteData("set_channel_welcome", "[welcome-message]", "Set the welcome message for the channel")
	setChannelWelcome.AddTextArgument("Welcome message for the channel", "[welcome-message]", "")
	welcomebot.AddCommand(setChannelWelcome)

	getChannelWelcome := model.NewAutocompleteData("get_channel_welcome", "", "Print the welcome message set for the channel")
	welcomebot.AddCommand(getChannelWelcome)

	deleteChannelWelcome := model.NewAutocompleteData("delete_channel_welcome", "", "Delete the welcome message for the channel")
	welcomebot.AddCommand(deleteChannelWelcome)

	return welcomebot
}
