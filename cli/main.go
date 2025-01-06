package main

import (
	"fmt"
	"log"
	"os"

	"github.com/antibomberman/dblayer"
	"github.com/antibomberman/dblayer/migrate"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dbl",
	Short: "Приложение для управления миграциями",
}
var migrationCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Команды для работы с миграциями",
}

var DownCmd = &cobra.Command{
	Use:   "down",
	Short: "Откатить последнюю миграцию",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Откат последней миграции...")
		// Здесь добавьте логику отката миграций
	},
}
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "generate default files",
	Run: func(cmd *cobra.Command, args []string) {
		migrate.InitDir()
		GenerateEnv()
	},
}
var CreateCmd = &cobra.Command{
	Use:   "create [название_миграции] ",
	Short: "Создать новую миграцию",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Необходимо указать название миграции")
			return
		}
		migrate.InitDir()
		GenerateEnv()

		dbl, err := dblayer.ConnectEnv()
		path, err := dbl.Migrate().CreateGo(args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Файл миграции создан: ", path)

	},
}
var UpCmd = &cobra.Command{
	Use:   "up",
	Short: "Применить все миграции",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Применение миграций...")
		dbl, err := dblayer.ConnectEnv()
		if err != nil {
			log.Fatal(err)
		}
		err = dbl.Migrate().Up()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func main() {
	rootCmd.AddCommand(migrationCmd)
	migrationCmd.AddCommand(CreateCmd)
	migrationCmd.AddCommand(UpCmd)
	migrationCmd.AddCommand(DownCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
