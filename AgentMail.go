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
	"io/ioutil"
	"net/http"

	"fmt"
	"github.com/crucibuild/agent-mail/schema"
	"github.com/crucibuild/sdk-agent-go/agentiface"
	"github.com/crucibuild/sdk-agent-go/agentimpl"
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
