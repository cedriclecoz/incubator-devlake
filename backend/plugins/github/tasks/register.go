/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import "github.com/apache/incubator-devlake/core/plugin"

var SubTaskMetaList []*plugin.SubTaskMeta

func RegisterSubtaskMeta(meta *plugin.SubTaskMeta) {
	SubTaskMetaList = append(SubTaskMetaList, meta)
}

// --- Secret Scanning Alert Subtasks Registration ---

var CollectSecretScanningAlertsMeta = plugin.SubTaskMeta{
	Name:             "Collect Secret Scanning Alerts",
	EntryPoint:       CollectSecretScanningAlerts,
	EnabledByDefault: true,
	Description:      "Collect secret scanning alerts from GitHub API.",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
	ProductTables:    []string{"github_api_secret_scanning_alerts"},
}

var ExtractSecretScanningAlertsMeta = plugin.SubTaskMeta{
	Name:             "Extract Secret Scanning Alerts",
	EntryPoint:       ExtractSecretScanningAlerts,
	EnabledByDefault: true,
	Description:      "Extract secret scanning alerts into domain model.",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
	ProductTables:    []string{"github_secret_scanning_alerts"},
}

func init() {
	RegisterSubtaskMeta(&CollectSecretScanningAlertsMeta)
	RegisterSubtaskMeta(&ExtractSecretScanningAlertsMeta)
}
