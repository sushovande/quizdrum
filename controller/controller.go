// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"quizdrum/model"
	"quizdrum/view"
)

// Controller holds state that is used across all handlers
type Controller struct {
	// P contains methods to store and retreive data.
	P *model.Persistence
	// V contains methods to display UI to the user.
	V *view.View
}
