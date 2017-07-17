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

package schema

import (
	"github.com/crucibuild/sdk-agent-go/agentiface"
	"github.com/crucibuild/sdk-agent-go/agentimpl"
	"github.com/crucibuild/sdk-agent-go/util"
)

// MailSentEvent is the go struct that reifies the mail-sent-event schema
type MailSentEvent struct {
	// nolint: golint, name should match name in schema
	Id string
}

// MailSentEventType holds the data type definition of the MailSentEvent type
var MailSentEventType agentiface.Type

func init() {
	t, err := util.GetStructType(&SendMailCommand{})
	if err != nil {
		panic(err)
	}
	MailSentEventType = agentimpl.NewTypeFromType("crucibuild/agent-mail#mail-sent-command", t)
}
