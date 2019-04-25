package vpa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDifference(t *testing.T) {
	for _, tc := range testDifferenceCases {
		res := difference(tc.testData1, tc.testData2)
		assert.Equal(t, res, tc.expected)
	}
}
