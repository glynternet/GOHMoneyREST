package main

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/glynternet/go-accounting/account"
	"github.com/glynternet/go-accounting/balance"
	"github.com/glynternet/go-money/currency"
	"github.com/glynternet/mon/internal/client"
	"github.com/glynternet/mon/pkg/filter"
	"github.com/glynternet/mon/pkg/storage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	appName = "montsv"

	keyServerHost   = "server-host"
	keyHistoricDays = "historic-days"
	keyForecastDays = "forecast-days"
)

func main() {
	err := cmdTSV.Execute()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

var now = time.Now()

var cmdTSV = &cobra.Command{
	Use:  appName,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.Client(viper.GetString(keyServerHost))
		as, err := c.SelectAccounts()
		if err != nil {
			return errors.Wrap(err, "selecting accounts")
		}

		var times []time.Time
		for i := -viper.GetInt(keyHistoricDays); i <= viper.GetInt(keyForecastDays); i++ {
			times = append(times, now.Add(time.Hour*24*time.Duration(i)))
		}
		if len(times) == 0 {
			return errors.New("date range yielded no dates")
		}

		*as = filter.AccountCondition(filter.AccountConditions{
			// must have existed by the last time of the generated times
			filter.Existed(times[len(times)-1]),

			// must have bit been closed by the first time of the generated times
			func(a storage.Account) bool {
				closedBeforeFirstTime := a.Account.Closed().Valid && a.Account.Closed().Time.Before(times[0])
				return !closedBeforeFirstTime
			},
		}.And).Filter(*as)

		// stored account balances
		var abss []AccountBalances
		for _, a := range *as {
			sbs, err := c.SelectAccountBalances(a.ID)
			if err != nil {
				return errors.Wrap(err, "selecting balances for account")
			}
			var bs = sbs.InnerBalances()
			abss = append(abss, AccountBalances{
				Account:  a.Account,
				Balances: bs,
			})
		}

		futures := filterTimesAfter(time.Now(), times...)

		// generated account balances
		gabss, err := generatedAccountBalances(futures)
		if err != nil {
			return errors.Wrap(err, "getting recurring costs accounts")
		}
		abss = append(abss, gabss...)

		datedBalances := [][]string{makeHeader(abss)}

		for _, t := range times {
			datedBalances = append(datedBalances, makeRow(t, abss))
		}

		w := csv.NewWriter(os.Stdout)
		w.Comma = '\t'

		return errors.Wrap(w.WriteAll(datedBalances), "writing separated values")
	},
}

func filterTimesAfter(t time.Time, times ...time.Time) []time.Time {
	var fs []time.Time
	for _, tt := range times {
		if tt.After(t) {
			fs = append(fs, tt)
		}
	}
	return fs
}

func generatedAccountBalances(times []time.Time) ([]AccountBalances, error) {
	var abss []AccountBalances
	for details, ag := range getAmountGenerators() {
		abs, err := generateAccountBalances(details, ag, times)
		if err != nil {
			return nil, errors.Wrap(err, "generating AccountBalances")
		}
		abss = append(abss, abs)
	}
	return abss, nil
}

func generateAccountBalances(ds accountDetails, ag amountGenerator, times []time.Time) (AccountBalances, error) {
	cc, err := currency.NewCode(ds.currencyString)
	if err != nil {
		return AccountBalances{}, errors.Wrapf(err, "creating new currency code")
	}

	a, err := account.New(ds.name, *cc, time.Time{}) // time/date of account is not used currently
	if err != nil {
		return AccountBalances{}, errors.Wrap(err, "creating new account")
	}

	var bs balance.Balances
	for _, t := range times {
		b, err := generateBalance(ag, t)
		if err != nil {
			return AccountBalances{}, errors.Wrapf(err, "generating balance for time:%s", t)
		}
		bs = append(bs, *b)
	}
	return AccountBalances{
		Account:  *a,
		Balances: bs,
	}, nil
}

func generateBalance(ag amountGenerator, at time.Time) (*balance.Balance, error) {
	b, err := balance.New(at, balance.Amount(ag.generateAmount(at)))
	return b, errors.Wrap(err, "creating balance")
}

func init() {
	cobra.OnInitialize(initConfig)
	cmdTSV.Flags().StringP(keyServerHost, "H", "", "server host")
	cmdTSV.Flags().Int(keyHistoricDays, 90, "days either side of now to provide data for")
	cmdTSV.Flags().Int(keyForecastDays, 30*6, "days in the future to provide data for")
	err := viper.BindPFlags(cmdTSV.Flags())
	if err != nil {
		log.Fatal(errors.Wrap(err, "binding root command flags"))
	}
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match
}

func makeHeader(accounts []AccountBalances) []string {
	hs := []string{"date"}
	for _, a := range accounts {
		hs = append(hs, a.Name())
	}
	hs = append(hs, "total")
	return hs
}

func makeRow(date time.Time, abss []AccountBalances) []string {
	dateString := date.Format("20060102")
	row := []string{dateString}
	f := filter.BalanceNot(filter.BalanceAfter(date))
	var total int
	for _, abs := range abss {
		amount := f.Filter(abs.Balances).Sum()
		row = append(row, strconv.Itoa(amount))
		total += amount
	}
	row = append(row, strconv.Itoa(total))
	return row
}

type AccountBalances struct {
	account.Account
	balance.Balances
}

type accountDetails struct {
	name           string
	currencyString string
}

func getAmountGenerators() map[accountDetails]amountGenerator {
	return map[accountDetails]amountGenerator{
		{
			name:           "daily spending",
			currencyString: "GBP",
		}: dailyRecurringAmount{
			Amount: -1500,
		},
		{
			name:           "bills",
			currencyString: "GBP",
		}: dailyRecurringAmount{
			Amount: -322, // 10000 a month but not on any specific day, as far as I know
		},
		{
			name:           "storage",
			currencyString: "EUR",
		}: monthlyRecurringCost{
			amount:      -7900,
			dateOfMonth: 1,
		},
		{
			name:           "health insurance",
			currencyString: "EUR",
		}: monthlyRecurringCost{
			amount:      -10250,
			dateOfMonth: 27,
		},
		{
			name:           "energy bill",
			currencyString: "EUR",
		}: monthlyRecurringCost{
			amount:      -3150,
			dateOfMonth: 12,
		},
		{
			name:           "ABN Amro bank account",
			currencyString: "EUR",
		}: monthlyRecurringCost{
			amount:      -155, //every 6 weeks
			dateOfMonth: 19,
		},
		{
			name:           "ABN Maandpremie",
			currencyString: "EUR",
		}: monthlyRecurringCost{
			amount:      -1461,
			dateOfMonth: 3,
		},
		{
			name:           "O2 Phone Bill",
			currencyString: "GBP",
		}: monthlyRecurringCost{
			amount:      -3000,
			dateOfMonth: 17,
		},
		{
			name:           "John & Emily Registration",
			currencyString: "EUR",
		}: dailyRecurringAmount{
			Amount: -142, // =-1000/7, Tenner a week
		},
		// {
		// 	name: "glynternet.com domain",
		// 	currencyString: "GBP",
		// }: yearlyRecurringAmount{
		// 	Date: {
		// 		Month: 10,
		// 		Day: 14
		// 	},
		// 	Amount:-1299,
		// },
	}
}
