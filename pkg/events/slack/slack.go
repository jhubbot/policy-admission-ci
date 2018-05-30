/*
Copyright 2018 Home Office All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/UKHomeOffice/policy-admission/pkg/api"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

type slackEvents struct {
	// name is the name of the kubernetes cluster
	name string
	// webhook is the url to send the events
	webhook string
}

// New creates and returns a slack sink
func New(cluster, webhook string) (api.Sink, error) {
	if _, err := url.Parse(webhook); err != nil {
		return nil, err
	}

	return &slackEvents{name: cluster, webhook: webhook}, nil
}

// Send sends the event into slack
func (s *slackEvents) Send(o metav1.Object, detail string) error {
	message := &messagePayload{
		Attachments: []*attachment{
			{
				Color: "#8B0000",
				Title: "Policy Admission - Denial (" + s.name + ")",
				Fields: []*attachmentField{
					{
						Title: "Detail",
						Value: detail,
					},
					{
						Title: "Namespace",
						Value: o.GetNamespace(),
						Short: true,
					},
					{
						Title: "UID",
						Value: string(o.GetUID()),
						Short: true,
					},
				},
				TimeStamp: time.Now().Unix(),
			},
		},
	}

	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := client.Post(s.webhook, "application/json", bytes.NewReader(encoded))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		status, _ := ioutil.ReadAll(resp.Body)

		return errors.New(string(status))
	}

	return nil
}
