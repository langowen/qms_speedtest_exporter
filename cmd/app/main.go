package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	// Запускаем команду с аргументами
	cmd := exec.Command("./qms_lib", "-O data/test.json", "-F json")

	// Получаем pipe для stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return
	}

	// Запускаем процесс
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		return
	}

	// Читаем вывод построчно
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Output: %s\n", line)

		// Здесь можно парсить вывод
		if strings.Contains(line, "test") {
			fmt.Println("Found 'test' in output!")
		}
	}

	// Ждем завершения процесса
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Error waiting for command: %v\n", err)
	}
}
