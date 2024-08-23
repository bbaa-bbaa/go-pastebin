// Copyright 2024 bbaa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

type Logger struct {
	Scope string
}

func (l *Logger) Println(scope string, a ...any) (n int, err error) {
	return fmt.Printf("[%s] %s %s", scope, time.Now().Format(time.RFC3339), strings.TrimRight(fmt.Sprint(a...), "\r\n")+"\r\n")
}

func (l *Logger) Error(a ...any) (n int, err error) {
	return l.Println(color.RedString(l.Scope), a...)
}

func (l *Logger) Info(a ...any) (n int, err error) {
	return l.Println(color.CyanString(l.Scope), a...)
}

func (l *Logger) Warning(a ...any) (n int, err error) {
	return l.Println(color.YellowString(l.Scope), a...)
}
