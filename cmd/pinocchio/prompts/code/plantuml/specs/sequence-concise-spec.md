PlantUML sequence diagrams are a powerful tool for visualizing interactions between different entities or participants. They use a simple syntax to denote messages sent between participants (`->`), with optional features like dotted arrows (`-->`) for distinct visualization, and reverse arrows (`<-` or `<--`) for improved readability.

Example:
```plantuml
@startuml
Alice -> Bob: Authentication Request
Bob --> Alice: Authentication Response
Alice <- Bob: Another authentication Response
@enduml
```

Participants can be declared using the `participant` keyword, allowing more control over their display order and visual representation. Different keywords like `actor`, `boundary`, `control`, etc., can be used to change the shape of the participant.

Example:
```plantuml
@startuml
participant Participant as Foo
actor Actor as Foo1
Foo -> Foo1 : To actor  
@enduml
```

You can rename a participant using the `as` keyword and change the background color of an actor or participant.

Example:
```plantuml
@startuml
actor Bob #red
participant Alice
Alice->Bob: Authentication Request
Bob->Alice: Authentication Response 
@enduml
```

Messages can be sent to oneself and can be multiline using `\n`.

Example:
```plantuml
@startuml
Alice -> Alice: This is a signal to self.\nIt also demonstrates\nmultiline \ntext
@enduml
```

You can change the arrow style and color for better visualization.

Example:
```plantuml
@startuml
Bob ->x Alice
Bob -[#red]> Alice : hello
@enduml
```

The `autonumber` keyword is used to automatically add an incrementing number to messages.

Example:
```plantuml
@startuml
autonumber
Bob -> Alice : Authentication Request
Bob <- Alice : Authentication Response  
@enduml
```

You can add a title to the page and display headers and footers using `header` and `footer`.

Example:
```plantuml
@startuml
header Page Header  
footer Page %page% of %lastpage%
title Example Title
Alice -> Bob : message 1
Alice -> Bob : message 2
@enduml
```

You can group messages together using keywords like `alt/else`, `opt`, `loop`, `par`, `break`, `critical`, and `group`.

Example:
```plantuml
@startuml
Alice -> Bob: Authentication Request
alt successful case
    Bob -> Alice: Authentication Accepted
else some kind of failure
    Bob -> Alice: Authentication Failure
end
@enduml
```

Notes can be added to messages or participants for additional information.

Example:
```plantuml
@startuml
Alice->Bob : hello
note left: this is a first note
Bob->Alice : ok 
note right: this is another note
@enduml
```

You can split a diagram using `==` separator to divide your diagram into logical steps.

Example:
```plantuml
@startuml
== Initialization ==
Alice -> Bob: Authentication Request
Bob --> Alice: Authentication Response
== Repetition ==
Alice -> Bob: Another authentication Request
Alice <-- Bob: Another authentication Response
@enduml
```

You can use reference in a diagram, using the keyword `ref over`.

Example:
```plantuml
@startuml
participant Alice
actor Bob 
ref over Alice, Bob : init
Alice -> Bob : hello
@enduml
```