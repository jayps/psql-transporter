package ui

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
)

func Select(label string, options []string) (string, error) {
	var out string
	prompt := &survey.Select{Message: label, Options: options}
	err := survey.AskOne(prompt, &out)
	return out, err
}

func ConfirmDanger(msg string) (bool, error) {
	var ok bool
	prompt := &survey.Confirm{Message: msg, Default: false}
	err := survey.AskOne(prompt, &ok)
	return ok, err
}

func StepSpinner[T any](title string, fn func() (T, error)) (T, error) {
	spinner, _ := pterm.DefaultSpinner.Start(title)
	res, err := fn()
	if err != nil {
		spinner.Fail(fmt.Sprintf("%s: %v", title, err))
		return res, err
	}
	spinner.Success(title)
	return res, nil
}

func ProgressSteps(steps []string, do func(update func(int))) {
	p, _ := pterm.DefaultProgressbar.WithTotal(len(steps)).
		WithTitle("Working...").
		WithRemoveWhenDone(false).
		Start()

	for _, s := range steps {
		p.UpdateTitle(s)
		do(func(done int) {})
		p.Increment()
	}

	p.Stop() // properly stop the bar
	pterm.Success.Println("All steps completed successfully!")
}

type Step struct {
	Title string
	Run   func() error
}

// RunSteps runs each step with a spinner. Stops at first error.
func RunSteps(steps []Step) error {
	for _, s := range steps {
		if _, err := StepSpinner(s.Title, func() (any, error) {
			return nil, s.Run()
		}); err != nil {
			return err
		}
	}
	return nil
}
