package main

import (
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

const (
	botUsername    = "triviabot"
	botDisplayName = "Triva Bot"
	botDescription = "A bot account created by the Trivia Plugin."
)

// Plugin represents the trivia bot plugin
type Plugin struct {
	plugin.MattermostPlugin

	client *pluginapi.Client
	// botUserID of the created bot account.
	botUserID string
}

// OnActivate ensure the bot account exists
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	bot := &model.Bot{
		Username:    botUsername,
		DisplayName: botDisplayName,
		Description: botDescription,
	}
	botUserID, appErr := p.client.Bot.EnsureBot(bot)
	if appErr != nil {
		return errors.Wrap(appErr, "failed to ensure bot user")
	}
	p.botUserID = botUserID

	err := p.API.RegisterCommand(getCommand())
	if err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	return nil
}
