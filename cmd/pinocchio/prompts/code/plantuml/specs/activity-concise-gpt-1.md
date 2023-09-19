PlantUML's activity diagram allows you to visually represent workflows of stepwise activities and actions with support for choice, iteration and concurrency. It offers a new syntax that doesn't require Graphviz installation and is easier to maintain.

To define an activity, use a label that starts with `:` and ends with `;`. Activities are linked in the order they are defined. For example:

```plantuml
@startuml
:Hello world;
:This is defined on
several **lines**;
@enduml
```

You can denote the start and end of a diagram using `start` and `stop` or `end` keywords:

```plantuml
@startuml
start
:Hello world;
:This is defined on
several **lines**;
stop
@enduml
```

Conditional statements can be added using `if`, `then` and `else` keywords:

```plantuml
@startuml
start
if (Graphviz installed?) then (yes)
  :process all\ndiagrams;
else (no)
  :process only
  __sequence__ and __activity__ diagrams;
endif
stop
@enduml
```

You can use `repeat` and `repeat while` keywords to create loops:

```plantuml
@startuml
start
repeat
  :read data;
  :generate diagrams;
repeat while (more data?) is (yes)
->no;
stop
@enduml
```

Grouping of activities can be done using `group` or `partition`:

```plantuml
@startuml
start
group Initialization 
    :read config file;
    :init internal variable;
end group
group Running group
    :wait for user interaction;
    :print information;
end group
stop
@enduml
```

You can add notes using `note` keyword:

```plantuml
@startuml
start
:foo1;
note right
  This note is on several
  //lines// and can
  contain <b>HTML</b>
end note
stop
@enduml
```

You can specify a color for some activities:

```plantuml
@startuml
start
:starting progress;
#HotPink:reading configuration files
These files should be edited at this point!;
#AAAAAA:ending of the process;
@enduml
```

You can use parentheses to denote connector:

```plantuml
@startuml
:Some activity;
(A)
detach
(A)
:Other activity;
@enduml
```
