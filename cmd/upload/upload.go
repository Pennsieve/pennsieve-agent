/*
Copyright © 2022 University of Pennsylvania <support@server>>

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
package upload

import (
	"github.com/spf13/cobra"
)

var (
	bucket string
	prefix string
)

var UploadCmd = &cobra.Command{
	Use:   "upload [flags] [PATH] [...PATH]",
	Short: "Upload files to the Pennsieve platform.",
	Long:  `Upload files to the Pennsieve platform.`,
}

func init() {

	UploadCmd.AddCommand(CancelCmd)
	UploadCmd.AddCommand(ManifestCmd)

}
