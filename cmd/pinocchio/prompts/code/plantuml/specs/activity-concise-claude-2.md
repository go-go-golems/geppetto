Here is a concise and simplified markdown specification for plantuml activity diagrams:

# Activity Diagrams

Activity diagrams visualize the flow of actions in a process. They consist of activities linked together with arrows showing the sequence.

## Activities

Activities are denoted by `:` and a label, ending with `;`.

```plantuml
@startuml
:Hello world;
:This is on 
multiple lines;
@enduml
```

## Sequence

Activities are linked sequentially in order of definition.

```plantuml
@startuml
:Foo;
:Bar;
@enduml
``` 

Use `->` to add text on arrows.

```plantuml
@startuml
:Foo;
-> Insert text;
:Bar; 
@enduml
```

## Conditionals

Use `if`/`then`/`else` for conditionals.

```plantuml
@startuml
if (condition?) then
  :Then; 
else
  :Else;
endif  
@enduml
```

## Loops

`while` loops repeat until condition is false.

```plantuml
@startuml
while (condition)
  :Repeat;
endwhile
@enduml
```

## Partitions

Group activities using `partition` blocks.

```plantuml
@startuml
partition "Group" {
  :Foo;
  :Bar;
}
@enduml
``` 

## Swimlanes

Divide activities into lanes using `|` for swimlanes.

```plantuml
@startuml
|Swimlane1|
:Foo;
|Swimlane2|
:Bar;
@enduml
```

## Complete Example

```plantuml
@startuml
start

if (multiprocessor?) then (yes)
  partition "Parallel" {
    :Process A;
    :Process B;
  }
else (no)
  :Process A;
  :Process B;
endif

while (condition)
  :Repeat; 
endwhile

stop
@enduml
```
