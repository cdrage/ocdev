package machineoutput

import (
	"encoding/json"
	"fmt"

	"github.com/openshift/odo/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Kind is what kind we should use in the machine readable output
const Kind = "Error"

// APIVersion is the current API version we are using
const APIVersion = "odo.openshift.io/v1alpha1"

// Error for machine readable output error messages
type Error struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Message           string `json:"message"`
}

// Success same as above, but copy-and-pasted just in case
// we change the output in the future
type Success struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Message           string `json:"message"`
}

// OutputSuccess outputs a "successful" machine-readable output format in json
func OutputSuccess(machineOutput Success) {
	printableOutput, err := json.Marshal(machineOutput)

	// If we error out... there's no way to output it (since we disable logging when using -o json)
	if err != nil {
		fmt.Fprintf(log.GetStderr(), "Unable to unmarshal JSON: %s\n", err.Error())
	} else {
		fmt.Fprintf(log.GetStdout(), "%s\n", string(printableOutput))
	}
}

// OutputError outputs a "successful" machine-readable output format in json
func OutputError(machineOutput Error) {
	printableOutput, err := json.Marshal(machineOutput)

	// If we error out... there's no way to output it (since we disable logging when using -o json)
	if err != nil {
		fmt.Fprintf(log.GetStderr(), "Unable to unmarshal JSON: %s\n", err.Error())
	} else {
		fmt.Fprintf(log.GetStderr(), "%s\n", string(printableOutput))
	}
}
