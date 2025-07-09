%{
package configparser

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/plakar/scheduler"
)

func Parser(lexer yyLexer) *ConfigParser {
	return lexer.(*ConfigParser)
}

%}

%union {
        b         bool
        s         string
	i         int64
	task      scheduler.Task
	schedule  scheduler.Schedule
	schedules []scheduler.Schedule
	times     []scheduler.Time
	strings   []string
	size      float64
	duration  time.Duration
	time      scheduler.Time
	datemask  scheduler.DateMask
	month     time.Month
	err       error
}

%token REPORTING ON OFF
%token BACKUP CHECK RESTORE SYNC MAINTENANCE
%token NAME CATEGORY ENVIRONMENT PERIMETER JOB TAG LATEST BEFORE SINCE
%token EXCLUDE
%token RETENTION

%token FROM TO WITH AND

%token DAY
%token AT EVERY FROM UNTIL

%token <i>        INTEGER
%token <s>        STRING REFERENCE
%token <err>      ERROR

%token <size>     SIZE
%token <duration> DURATION
%token <time>     TIME
%token <datemask> WEEKDAY
%token <month>    MONTH

%type <schedule>  schedule_time schedule
%type <schedules> schedule_list
%type <task>      task
%type <size>      size
%type <duration>  duration
%type <time>      time from_time until_time
%type <times>     time_list
%type <datemask>  weekday weekday_list schedule_days
%type <month>     month

%type <b>         on_off
%type <strings>   source_list
%type <strings>   string_list
%type <s>         repository
%type <s>         source
%type <s>         destination
%type <s>         sync_direction

%start grammar

%%

grammar:
| grammar reporting
| grammar job
;

reporting:
  REPORTING on_off { Parser(yylex).reporting = $2 }
;

job:
  JOB STRING task schedule_list {
    if Parser(yylex).HasJob($2) {
       Parser(yylex).Error(fmt.Sprintf("job %q already defined", $2))
	 goto ret1
    }
    Parser(yylex).PushJob($2, $3, $4)
}
;

task:
  BACKUP source TO repository {
    Parser(yylex).MakeBackupTask($4, $2)
  } backup_opts { $$ = Parser(yylex).currentTask }
| CHECK repository {
    $$ = Parser(yylex).MakeCheckTask($2)
  }
| MAINTENANCE ON repository {
    Parser(yylex).MakeMaintenanceTask($3)
  } maintenance_opts { $$ = Parser(yylex).currentTask }
| RESTORE repository TO destination {
    $$ = Parser(yylex).MakeRestoreTask($2, $4)
  }
| SYNC repository sync_direction repository {
    $$ = Parser(yylex).MakeSyncTask($2, $3, $4)
  }
;

source:
  STRING    { $$ = $1 }
| REFERENCE { $$ = $1 }
;

source_list:
  source { $$ = append(make([]string, 0, 5), $1) }
| source_list source { $$ = append($1, $2) }
;

destination:
  STRING    { $$ = $1 }
| REFERENCE { $$ = $1 }
;

repository:
  STRING    { $$ = $1 }
| REFERENCE { $$ = $1 }
;

backup_opts:
| backup_opts backup_opt
;

backup_opt:
  TAG string_list {
    task := Parser(yylex).currentTask.(*scheduler.BackupTask)
    task.Cmd.Tags = $2
  }
| EXCLUDE STRING {
    task := Parser(yylex).currentTask.(*scheduler.BackupTask)
    task.Cmd.Excludes = append(task.Cmd.Excludes, $2)
  }
| RETENTION duration {
    task := Parser(yylex).currentTask.(*scheduler.BackupTask)
    task.Retention = $2
  }
;

maintenance_opts:
| maintenance_opts maintenance_opt
;

maintenance_opt:
  RETENTION duration {
    task := Parser(yylex).currentTask.(*scheduler.MaintenanceTask)
    task.Retention = $2
  }
;

sync_direction:
  TO { $$ = "to" }
| FROM { $$ = "from" }
| WITH { $$ = "with" }
;

schedule_list:
  schedule { $$ = append($$, $1) }
| schedule_list schedule { $$ = append($1, $2) }
;

schedule:
  schedule_time schedule_days {
    $$ = $1.WithDateMask($2)
  }
;

schedule_time:
  AT time_list {
    $$ = Parser(yylex).MakeScheduleAt($2)
  }
| EVERY duration from_time until_time {
    $$ = Parser(yylex).MakeScheduleEvery($2, $3, $4)
  }
;

from_time:
  { $$ = scheduler.UndefinedTime }
| FROM time { $$ = $2 }
;

until_time:
  { $$ = scheduler.UndefinedTime }
| UNTIL time { $$ = $2 }
;

schedule_days:
  { $$ = scheduler.EveryDay }
| ON weekday_list { $$ = scheduler.EveryDay.SetWeekdayMask($2) }
;

weekday_list:
  weekday { $$ = $1 }
| weekday_list ',' weekday { $$ = $1 | $3 }
;

time_list:
  time { $$ = append(make([]scheduler.Time, 0, 1), $1) }
| time_list ',' time { $$ = append($1, $3) }
;

string_list:
  STRING { $$ = append(make([]string, 0, 1), $1) }
| string_list ',' STRING { $$ = append($1, $3) }
;

size:
  SIZE { $$ = $1 }
;

duration:
  DURATION { $$ = $1 }
;

time:
  TIME { $$ = $1 }
;

weekday:
  WEEKDAY { $$ = $1 }
;

month:
  MONTH { $$ = $1 }
;

on_off:
  ON { $$ = true }
| OFF { $$ = false }
;
