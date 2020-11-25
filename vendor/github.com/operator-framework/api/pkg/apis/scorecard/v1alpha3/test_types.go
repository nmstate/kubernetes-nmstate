package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// State is a type used to indicate the result state of a Test.
type State string

const (
	// PassState occurs when a Test's ExpectedPoints == MaximumPoints.
	PassState State = "pass"
	// FailState occurs when a Test's ExpectedPoints == 0.
	FailState State = "fail"
	// ErrorState occurs when a Test encounters a fatal error and the reported points should not be considered.
	ErrorState State = "error"
)

// TestResult contains the results of an individual scorecard test
type TestResult struct {
	// Name is the name of the test
	Name string `json:"name,omitempty"`
	// Log holds a log produced from the test (if applicable)
	Log string `json:"log,omitempty"`
	// State is the final state of the test
	State State `json:"state"`
	// Errors is a list of the errors that occurred during the test (this can include both fatal and non-fatal errors)
	Errors []string `json:"errors,omitempty"`
	// Suggestions is a list of suggestions for the user to improve their score (if applicable)
	Suggestions []string `json:"suggestions,omitempty"`
}

// TestStatus contains collection of testResults.
type TestStatus struct {
	Results []TestResult `json:"results,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Test specifies a single test run.
type Test struct {
	metav1.TypeMeta `json:",inline"`
	Spec            TestConfiguration `json:"spec,omitempty"`
	Status          TestStatus        `json:"status,omitempty"`
}

// TestList is a list of tests.
type TestList struct {
	metav1.TypeMeta `json:",inline"`
	Items           []Test `json:"items"`
}

func NewTest() Test {
	return Test{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "Test",
		},
	}
}

func NewTestList() TestList {
	return TestList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "TestList",
		},
	}
}
