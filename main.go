package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type exchangeResponse struct {
	Date string             `json:"date"`
	USD  map[string]float64 `json:"usd"`
}

type model struct {
	amount       string
	amountError  string
	from         string
	to           string
	currencies   []string
	selected     int
	choosingFrom bool
	parsedAmount float64
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "backspace":
			if len(m.amount) == 0 {
				return m, nil
			}
			m.amount = m.amount[:len(m.amount)-1]
		case "s":
			old := m.from
			m.from = m.to
			m.to = old
		case "up":
			if m.selected > 0 {
				m.selected--
			}
		case "down":
			if m.selected < len(m.currencies)-1 {
				m.selected++
			}
		case "tab":
			m.choosingFrom = !m.choosingFrom
		case "enter":
			if m.choosingFrom {
				m.from = m.currencies[m.selected]
			} else {
				m.to = m.currencies[m.selected]
			}
		default:
			key := msg.String()

			if strings.Contains("0123456789", key) && len(key) == 1 {
				m.amount += key
			} else if key == "." && !strings.Contains(m.amount, ".") {
				m.amount += key
			}

			if m.amount == "" {
				m.amountError = ""
			} else {
				_, err := parseAmount(m.amount)
				if err != nil {
					m.amountError = "Некорректная сумма"
				} else {
					m.amountError = ""
				}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	fmt.Fprintf(&b, "gcoin — Currency Converter\n\n")
	fmt.Fprintf(&b, "Сумма: %s\n", m.amount)

	if m.amountError != "" {
		fmt.Fprintf(&b, "Ошибка: %s\n", m.amountError)
	}

	if m.choosingFrom {
		fmt.Fprintf(&b, "Из валюты: %s <\n", m.from)
		fmt.Fprintf(&b, "В валюту: %s\n\n", m.to)
	} else {
		fmt.Fprintf(&b, "Из валюты: %s\n", m.from)
		fmt.Fprintf(&b, "В валюту: %s <\n\n", m.to)
	}
	//fmt.Fprintf(&b, "Из валюты: %s\n", m.from)
	//fmt.Fprintf(&b, "В валюту: %s\n\n", m.to)

	for i, currency := range m.currencies {
		if i == m.selected {
			fmt.Fprintf(&b, "> %s\n", currency)
		} else {
			fmt.Fprintf(&b, "  %s\n", currency)
		}
	}

	fmt.Fprintf(&b, "\nНажми q для выхода.\n")

	return b.String()
}

func parseAmount(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func main() {
	p := tea.NewProgram(model{
		from:       "USD",
		to:         "EUR",
		currencies: []string{"USD", "EUR", "KZT", "RUB", "GBP", "JPY"},
		selected:   0,
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

}
