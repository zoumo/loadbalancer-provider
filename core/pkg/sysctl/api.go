/*
Copyright 2017 Caicloud authors. All rights reserved.

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

package sysctl

// BulkModify changes the settings according to the given sysctlAdjustments,
// returns the original value and error
func BulkModify(sysctlAdjustments map[string]string) (originalSysctl map[string]string, err error) {
	originalSysctl = make(map[string]string)
	sys := New()
	for k, v := range sysctlAdjustments {
		defVar, err := sys.GetSysctl(k)
		if err != nil {
			return originalSysctl, err
		}
		originalSysctl[k] = defVar

		if err := sys.SetSysctl(k, v); err != nil {
			return originalSysctl, err
		}
	}
	return originalSysctl, nil
}
