// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

// Namespace qualifies for names used to describe spans and metrics.
type Namespace string

// Name pairs a Namespace and a Base name.  OpenTelemetry libraries
// will presume that identical names refer to the same thing; using
// namespaces offers a way to disambiguate names used by different
// modules of code.
type Name struct {
	Namespace Namespace
	Base      string
}

func (n Namespace) Name(base string) Name {
	return Name{
		Namespace: n,
		Base:      base,
	}
}

func (n Name) String() string {
	if n.Namespace == "" {
		return n.Base
	}
	return string(n.Namespace) + "/" + n.Base
}
