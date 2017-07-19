// Copyright (C) 2016 Christophe Camel, Jonathan Pigrée
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/crucibuild/agent-mail/schema"
	"github.com/crucibuild/sdk-agent-go/agentiface"
	"github.com/crucibuild/sdk-agent-go/agentimpl"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"net/url"
)

const (
	// OptionMailServerKey is the name of the option for the mail server
	OptionMailServerKey = "mailserver"
	// OptionMailServerDefaultValue is the default URI of the mail server
	OptionMailServerDefaultValue = "smpt://localhost:25/"
	// OptionMailServerUsage is the documentation for the option mail server
	OptionMailServerUsage = "Specifies the URI (with login credentials) of the mail server to use. The typical form is following (taking SMTP as an example): smtp://[username@]host[:port][?password=somepwd]. By default, the local smtp server (available on port 25) without any authentication is used."
)

// Resources represents an handler on the various data files
// Used by the agent(avro files, manifest, etc...).
var Resources http.FileSystem

// AgentMail is an implementation over the Agent implementation
// available in sdk-agent-go.
type AgentMail struct {
	*agentimpl.Agent
}

// nolint: unparam
func mustOpenResources(path string) []byte {
	file, err := Resources.Open(path)

	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadAll(file)

	if err != nil {
		panic(err)
	}

	return content
}

// NewAgentMail creates a new instance of AgentMail.
func NewAgentMail() (agentiface.Agent, error) {
	var agentSpec map[string]interface{}

	manifest := mustOpenResources("/resources/manifest.json")

	err := json.Unmarshal(manifest, &agentSpec)

	if err != nil {
		return nil, err
	}

	impl, err := agentimpl.NewAgent(agentimpl.NewManifest(agentSpec))

	if err != nil {
		return nil, err
	}

	agent := &AgentMail{
		impl,
	}

	if err := agent.init(); err != nil {
		return nil, err
	}

	return agent, nil
}

func (a *AgentMail) init() (err error) {
	// register configuration options
	a.SetDefaultConfigOption(OptionMailServerKey, OptionMailServerDefaultValue)

	// registers additional CLI options
	for _, c := range a.Cli.RootCommand().Commands() {
		if c.Use == "agent:start" {
			c.Flags().String(OptionMailServerKey, OptionMailServerDefaultValue, OptionMailServerUsage)
			a.BindConfigPFlag(OptionMailServerKey, c.Flags().Lookup(OptionMailServerKey)) // nolint: errcheck, no error can occur here, by construction.

			break
		}
	}

	// register schemas:
	schemas := []string{
		"/schema/send-mail-command.avro",
		"/schema/mail-sent-event.avro",
	}
	if err = a.registerSchemas(schemas); err != nil {
		return err
	}

	// register types:
	types := []agentiface.Type{
		schema.SendMailCommandType,
		schema.MailSentEventType,
	}
	err = a.registerTypes(types)

	// register state callback
	a.RegisterStateCallback(a.onStateChange)

	return err
}

func (a *AgentMail) registerSchemas(pathes []string) error {
	for _, path := range pathes {
		content := mustOpenResources(path)

		s, err := agentimpl.LoadAvroSchema(string(content[:]), a)
		if err != nil {
			return fmt.Errorf("Failed to load schema %s: %s", path, err.Error())
		}

		_, err = a.SchemaRegister(s)

		if err != nil {
			return fmt.Errorf("Failed to register schema %s: %s", path, err.Error())
		}
	}
	return nil
}

func (a *AgentMail) registerTypes(types []agentiface.Type) error {
	for _, t := range types {
		if _, err := a.TypeRegister(t); err != nil {
			return fmt.Errorf("Failed to register type %s (which is a %s): %s", t.Name(), t.Type().Name(), err.Error())
		}
	}
	return nil
}

func (a *AgentMail) onStateChange(state agentiface.State) error {
	switch state {
	case agentiface.StateConnected:
		if _, err := a.RegisterCommandCallback(agentiface.MessageName(schema.SendMailCommandType.Name()), a.onSendMailCommand); err != nil {
			return err
		}
	}
	return nil
}

func (a *AgentMail) onSendMailCommand(ctx agentiface.CommandCtx) error {
	cmd := ctx.Message().(*schema.SendMailCommand)

	a.Info(fmt.Sprintf("Received send-mail-command: From: '%s' To: '%s' Subject: '%s' ", cmd.From, cmd.To, cmd.Subject))

	// Connect to the remote SMTP server.
	c, err := a.connectToMailServer()

	if err != nil {
		return nil
	}

	// Set the sender and recipient first
	if err := c.Mail(cmd.From); err != nil {
		return err
	}

	if err := c.Rcpt(cmd.To); err != nil {
		return err
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(wc, cmd.Content)
	if err != nil {
		return err
	}

	err = wc.Close()
	if err != nil {
		return err
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		return err
	}

	return ctx.SendEvent(&schema.MailSentEvent{Id: cmd.Id})
}

func (a *AgentMail) connectToMailServer() (*smtp.Client, error) {
	mailserverURI, err := url.Parse(a.GetConfigString(OptionMailServerKey))

	if err != nil {
		return nil, err
	}

	if mailserverURI.User.Username() == "" {
		// use simple connection
		c, err := smtp.Dial(fmt.Sprintf("%s:%s", mailserverURI.Host, mailserverURI.Port()))
		if err != nil {
			return nil, err
		}
		return c, nil
	}

	return nil, fmt.Errorf("Authentication scheme not yet supported")

}
