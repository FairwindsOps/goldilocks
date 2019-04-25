package vpa

var empty []string

var testDifferenceCases = []struct {
	description string
	testData1   []string
	testData2   []string
	expected    []string
}{
	{
		description: "empty case",
		testData1:   []string{"a", "b", "c"},
		testData2:   []string{"a", "b", "c"},
		expected:    empty,
	},
	{
		description: "extra item on right",
		testData1:   []string{"a", "b"},
		testData2:   []string{"a", "b", "c"},
		expected:    empty,
	},
	{
		description: "extra item on left",
		testData1:   []string{"a", "b", "c"},
		testData2:   []string{"a", "b"},
		expected:    []string{"c"},
	},
}
