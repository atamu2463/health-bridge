package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"backend/config"

	"golang.org/x/term"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		log.Printf("ユーザー作成に失敗しました: %v", err)
		os.Exit(1)
	}
}

func run(args []string, stdin *os.File, stdout io.Writer) (runErr error) {
	flags := flag.NewFlagSet("create-user", flag.ContinueOnError)
	flags.SetOutput(stdout)
	name := flags.String("name", "", "ユーザー名")
	email := flags.String("email", "", "メールアドレス")
	role := flags.String("role", "", "manager または employee")
	managerEmail := flags.String("manager-email", "", "employeeを担当するmanagerのメールアドレス")

	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return errors.New("引数を確認してください")
	}

	password, err := readAndConfirmPassword(stdin, stdout)
	if err != nil {
		return err
	}

	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		return err
	}
	db, err := config.ConnectDB(databaseURL)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return errors.New("DBインスタンスを確認できませんでした")
	}
	defer func() {
		if err := sqlDB.Close(); err != nil && runErr == nil {
			runErr = errors.New("DB接続を終了できませんでした")
		}
	}()

	if _, err := createUser(db, createUserInput{
		Name:         *name,
		Email:        *email,
		Role:         *role,
		ManagerEmail: *managerEmail,
		Password:     password,
	}, passwordHashCost); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "ユーザーを作成しました")
	return nil
}

func readAndConfirmPassword(stdin *os.File, stdout io.Writer) (string, error) {
	if term.IsTerminal(int(stdin.Fd())) {
		password, err := readPasswordFromTerminal(stdin, stdout, "パスワード: ")
		if err != nil {
			return "", err
		}
		confirmation, err := readPasswordFromTerminal(stdin, stdout, "パスワード（確認）: ")
		if err != nil {
			return "", err
		}
		if password != confirmation {
			return "", errors.New("パスワードが一致しません")
		}
		return password, nil
	}

	reader := bufio.NewReader(stdin)
	password, err := readPasswordLine(reader, stdout, "パスワード: ")
	if err != nil {
		return "", err
	}
	confirmation, err := readPasswordLine(reader, stdout, "パスワード（確認）: ")
	if err != nil {
		return "", err
	}
	if password != confirmation {
		return "", errors.New("パスワードが一致しません")
	}
	return password, nil
}

func readPasswordFromTerminal(stdin *os.File, stdout io.Writer, prompt string) (string, error) {
	fmt.Fprint(stdout, prompt)
	password, err := term.ReadPassword(int(stdin.Fd()))
	fmt.Fprintln(stdout)
	if err != nil {
		return "", errors.New("パスワードを読み取れませんでした")
	}
	return string(password), nil
}

func readPasswordLine(reader *bufio.Reader, stdout io.Writer, prompt string) (string, error) {
	fmt.Fprint(stdout, prompt)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", errors.New("パスワードを読み取れませんでした")
	}
	return strings.TrimRight(value, "\r\n"), nil
}
