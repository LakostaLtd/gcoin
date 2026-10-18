package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type exchangeResponse struct {
	Date string             `json:"date"`
	USD  map[string]float64 `json:"usd"`
}

type model struct {
	amount             string
	amountError        string
	from               string
	to                 string
	currencies         []string
	selected           int
	choosingFrom       bool
	parsedAmount       float64
	rates              map[string]float64
	rateDate           string
	width              int
	height             int
	offset             int
	searching          bool
	searchQuery        string
	filteredCurrencies []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc":
				m.searching = false
				m.searchQuery = ""
				m.filteredCurrencies = nil
				m.selected = 0
				m.offset = 0
				return m, nil

			case "enter":
				if len(m.filteredCurrencies) > 0 &&
					m.selected >= 0 &&
					m.selected < len(m.filteredCurrencies) {
					if m.choosingFrom {
						m.from = m.filteredCurrencies[m.selected]
					} else {
						m.to = m.filteredCurrencies[m.selected]
					}
				}

				m.searching = false
				m.searchQuery = ""
				m.filteredCurrencies = nil
				m.selected = 0
				m.offset = 0
				return m, nil

			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				} else {
					m.searching = false
				}

			case "up":
				if m.selected > 0 {
					m.selected--
				}

			case "down":
				if m.selected < len(m.filteredCurrencies)-1 {
					m.selected++
				}

			default:
				key := msg.String()
				if len(key) == 1 &&
					((key[0] >= 'a' && key[0] <= 'z') ||
						(key[0] >= 'A' && key[0] <= 'Z')) {
					m.searchQuery += strings.ToUpper(key)
				}
			}

			m.filteredCurrencies = nil
			for _, currency := range m.currencies {
				if strings.HasPrefix(currency, m.searchQuery) {
					m.filteredCurrencies = append(m.filteredCurrencies, currency)
				}
			}

			if m.selected >= len(m.filteredCurrencies) {
				m.selected = len(m.filteredCurrencies) - 1
			}
			if m.selected < 0 {
				m.selected = 0
			}

			m.offset = 0
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "f":
			m.searching = true
			m.choosingFrom = true
			m.searchQuery = ""
			m.filteredCurrencies = m.currencies
			m.selected = 0
			m.offset = 0
			return m, nil

		case "t":
			m.searching = true
			m.choosingFrom = false
			m.searchQuery = ""
			m.filteredCurrencies = m.currencies
			m.selected = 0
			m.offset = 0
			return m, nil

		case "backspace":
			if len(m.amount) > 0 {
				m.amount = m.amount[:len(m.amount)-1]
			}

		case "s":
			old := m.from
			m.from = m.to
			m.to = old

		case "up":
			if m.selected > 0 {
				m.selected--
			}
			if m.selected < m.offset {
				m.offset = m.selected
			}

		case "down":
			if m.selected < len(m.currencies)-1 {
				m.selected++
			}

			visible := m.height - 12
			if visible < 1 {
				visible = 1
			}
			if m.selected >= m.offset+visible {
				m.offset = m.selected - visible + 1
			}

		case "tab":
			m.choosingFrom = !m.choosingFrom

		case "enter":
			if len(m.currencies) > 0 &&
				m.selected >= 0 &&
				m.selected < len(m.currencies) {
				if m.choosingFrom {
					m.from = m.currencies[m.selected]
				} else {
					m.to = m.currencies[m.selected]
				}
			}

		default:
			key := msg.String()

			if len(key) == 1 &&
				key[0] >= '0' && key[0] <= '9' {
				m.amount += key
			} else if key == "." && !strings.Contains(m.amount, ".") {
				m.amount += key
			}
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

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, "gcoin — Currency Converter")
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "Сумма: %s\n", m.amount)

	if m.choosingFrom {
		fmt.Fprintf(&b, "Из валюты: %s  <\n", m.from)
		fmt.Fprintf(&b, "В валюту:  %s\n", m.to)
	} else {
		fmt.Fprintf(&b, "Из валюты: %s\n", m.from)
		fmt.Fprintf(&b, "В валюту:  %s  <\n", m.to)
	}

	fmt.Fprintln(&b)

	if m.amount != "" {
		amount, err := parseAmount(m.amount)
		if err == nil {
			result, err := convert(amount, m.from, m.to, m.rates)
			if err == nil {
				fmt.Fprintf(&b, "Результат: %.2f %s\n", result, m.to)
			} else {
				fmt.Fprintf(&b, "Ошибка конвертации: %v\n", err)
			}
		} else {
			fmt.Fprintln(&b, "Некорректная сумма")
		}
	} else {
		fmt.Fprintln(&b, "Введи сумму для конвертации")
	}

	fmt.Fprintf(&b, "Дата курса: %s\n", m.rateDate)

	if m.amountError != "" {
		fmt.Fprintf(&b, "Ошибка: %s\n", m.amountError)
	}

	fmt.Fprintln(&b)

	currencies := m.currencies
	if m.searching {
		currencies = m.filteredCurrencies
		fmt.Fprintf(&b, "Поиск валюты: %s_\n", m.searchQuery)

		if len(currencies) == 0 {
			fmt.Fprintln(&b, "Совпадений не найдено")
		}
	} else {
		fmt.Fprintln(&b, "Выбор валюты (↑/↓, Enter)")
		fmt.Fprintln(&b, "Поиск: f — исходная валюта, t — целевая валюта")
	}

	visible := m.height - 15
	if visible < 1 {
		visible = 1
	}

	end := m.offset + visible
	if end > len(currencies) {
		end = len(currencies)
	}

	for i := m.offset; i < end; i++ {
		if i == m.selected {
			fmt.Fprintf(&b, "> %s\n", currencies[i])
		} else {
			fmt.Fprintf(&b, "  %s\n", currencies[i])
		}
	}

	fmt.Fprintln(&b)

	if m.searching {
		fmt.Fprintln(&b, "↑/↓ — выбор | Enter — подтвердить | Backspace — удалить")
		fmt.Fprintln(&b, "Esc — отменить поиск | q — выход")
	} else {
		fmt.Fprintln(&b, "Tab — сменить поле | s — обменять валюты | q — выход")
	}

	return b.String()
}

func parseAmount(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func fetchRates() (*exchangeResponse, error) {
	resp, err := http.Get("https://latest.currency-api.pages.dev/v1/currencies/usd.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API вернул статус: %s", resp.Status)
	}

	var rates exchangeResponse

	// Читаем JSON из тела ответа и заполняем поля структуры
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return nil, err
	}

	return &rates, nil
}

func convert(amount float64, from, to string, rates map[string]float64) (float64, error) {
	from = strings.ToLower(from)
	to = strings.ToLower(to)

	if amount < 0 {
		return 0, fmt.Errorf("сумма не может быть отрицательной")
	}

	if from == to {
		return amount, nil
	}

	fromRate := 1.0
	if from != "usd" {
		var ok bool
		fromRate, ok = rates[from]
		if !ok || fromRate <= 0 {
			return 0, fmt.Errorf("курс валюты %s не найден", strings.ToUpper(from))
		}
	}

	toRate := 1.0
	if to != "usd" {
		var ok bool
		toRate, ok = rates[to]
		if !ok || toRate <= 0 {
			return 0, fmt.Errorf("курс валюты %s не найден", strings.ToUpper(to))
		}
	}

	result := amount / fromRate * toRate

	return result, nil
}

func getCurrencies(rates map[string]float64) []string {
	currencies := make([]string, 0, len(rates))

	for currency := range rates {
		currencies = append(currencies, strings.ToUpper(currency))
	}

	currencies = append(currencies, "USD")
	sort.Strings(currencies)

	return currencies
}

func main() {
	rates, err := fetchRates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка получения курсов: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model{
		from:       "USD",
		to:         "EUR",
		currencies: getCurrencies(rates.USD),
		selected:   0,
		rates:      rates.USD,
		rateDate:   rates.Date,
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

}
