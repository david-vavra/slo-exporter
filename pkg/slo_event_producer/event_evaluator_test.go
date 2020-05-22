//revive:disable:var-naming
package slo_event_producer

//revive:enable:var-naming

import (
	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.seznam.net/sklik-devops/slo-exporter/pkg/event"
	"gitlab.seznam.net/sklik-devops/slo-exporter/pkg/stringmap"
	"testing"
)

type sloEventTestCase struct {
	inputEvent        event.HttpRequest
	expectedSloEvents []event.Slo
	rulesConfig       rulesConfig
}

func TestSloEventProducer(t *testing.T) {
	testCases := []sloEventTestCase{
		{
			inputEvent: event.HttpRequest{Metadata: stringmap.StringMap{"statusCode": "502"}, SloClassification: &event.SloClassification{Class: "class", App: "app", Domain: "domain"}},
			rulesConfig: rulesConfig{Rules: []ruleOptions{
				{
					SloMatcher:                       sloMatcher{Domain: "domain"},
					MetadataMatcherConditionsOptions: []operatorOptions{},
					FailureConditionsOptions: []operatorOptions{
						operatorOptions{
							Operator: "numberHigherThan", Key: "statusCode", Value: "500",
						},
					},
					AdditionalMetadata: stringmap.StringMap{"slo_type": "availability"},
				},
			},
			},
			expectedSloEvents: []event.Slo{
				{Domain: "domain", Class: "class", App: "app", Key: "", Metadata: stringmap.StringMap{"slo_type": "availability"}, Result: event.Fail},
			},
		},
		{
			inputEvent: event.HttpRequest{Metadata: stringmap.StringMap{"statusCode": "200"}, SloClassification: &event.SloClassification{Class: "class", App: "app", Domain: "domain"}},
			rulesConfig: rulesConfig{Rules: []ruleOptions{
				{
					SloMatcher:                       sloMatcher{Domain: "domain"},
					MetadataMatcherConditionsOptions: []operatorOptions{},
					FailureConditionsOptions: []operatorOptions{
						operatorOptions{
							Operator: "numberHigherThan", Key: "statusCode", Value: "500",
						},
					},
					AdditionalMetadata: stringmap.StringMap{"slo_type": "availability"},
				},
			},
			},
			expectedSloEvents: []event.Slo{
				{Domain: "domain", Class: "class", App: "app", Key: "", Metadata: stringmap.StringMap{"slo_type": "availability"}, Result: event.Success},
			},
		},
	}

	for _, tc := range testCases {
		out := make(chan *event.Slo, 100)
		testedEvaluator, err := NewEventEvaluatorFromConfig(&tc.rulesConfig, logrus.New())
		if err != nil {
			t.Errorf("error when loading config: %v error: %v", tc.rulesConfig, err)
		}
		testedEvaluator.Evaluate(&tc.inputEvent, out)
		close(out)
		var results []event.Slo
		for newEvent := range out {
			results = append(results, *newEvent)
		}
		if diff := deep.Equal(tc.expectedSloEvents, results); diff != nil {
			t.Errorf("events are different %+v, \nexpected: %+v\n result: %+v\n input event metadata: %+v", diff, tc.expectedSloEvents, results, tc.inputEvent.Metadata)
		}
	}
}

type getMetricsFromRuleOptionsTestCase struct {
	Name           string
	RulesConfig    rulesConfig
	ExpectedMetric []metric
}

func TestConfig_getMetricsFromRuleOptions(t *testing.T) {
	testCases := []getMetricsFromRuleOptionsTestCase{
		{
			Name: "Configured rules are exposed as Prometheus metric",
			RulesConfig: rulesConfig{[]ruleOptions{
				{
					MetadataMatcherConditionsOptions: []operatorOptions{
						{
							Operator: "equalTo",
							Key:      "name",
							Value:    "ad.banner",
						},
					},
					SloMatcher: sloMatcher{},
					FailureConditionsOptions: []operatorOptions{
						{
							Operator: "numberHigherThan",
							Key:      "prometheusQueryResult",
							Value:    "6300",
						},
						{
							Operator: "numberEqualOrLessThan",
							Key:      "prometheusQueryResult",
							Value:    "7000",
						},
					},
					AdditionalMetadata: stringmap.StringMap{"foo": "bar"},
					HonorSloResult:     false,
				},
			},
			},
			ExpectedMetric: []metric{
				{
					Labels: stringmap.StringMap{"foo": "bar", "name": "ad.banner", "operator": "numberHigherThan"},
					Value:  6300,
				},
				{
					Labels: stringmap.StringMap{"foo": "bar", "name": "ad.banner", "operator": "numberEqualOrLessThan"},
					Value:  7000,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.Name,
			func(t *testing.T) {
				var (
					metrics []metric
					err     error
				)
				evaluator, err := NewEventEvaluatorFromConfig(&testCase.RulesConfig, logrus.New())
				if err != nil {
					t.Error(err)
				}
				metrics, _, err = evaluator.getMetricsFromRuleOptions()
				if err != nil {
					t.Error(err)
				}
				assert.Equal(t, testCase.ExpectedMetric, metrics)
			},
		)
	}
}
