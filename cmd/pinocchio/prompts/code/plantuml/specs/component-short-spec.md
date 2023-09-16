Here is a short overview of the syntax for plantuml component diagrams:

# Components

```
[Component1]
[Component2] as Comp2
```

# Interfaces

```
() "Interface1"
() "Interface2" as Interf2
```

# Links

```
Comp2 ..> Interf2 : use
```

# Notes

```
note left of Interf2
This is a note
end note
```

# Grouping Components

```
package "Some Group" {
[Component2]
}
```

# Changing Arrow Direction

```
[Component1] -up-> [Component2]
[Component2] -right-> [Component1]
```

# Coloring Components

```
component [Web Server] #Yellow
```

# Ports

```
component Client {
port port1
port port2
}
[Component1] -> port1
[Component2] -> port2
```

# Examples

Here are some example diagrams to demonstrate the component diagram syntax:

```plantuml
@startuml

# Simple Diagram
[Component1] ..> [Component2]

@enduml
```

```plantuml
@startuml 

# Grouped Components Diagram
package "Billing System" {
  [Billing Component]
}

node "Reporting System" {
  [Report Component]  
}

[Billing Component] ..> [Report Component]

@enduml
```

```plantuml
@startuml

# Colored and Styled Components 

component [API] #Yellow
component [Server] #Green
database "Database" {
  [DB Component]
}

[API] -> [Server]
[Server] -> [DB Component] 

@enduml
```

```plantuml
@startuml

# Bidirectional Ports

() "Interface1"
() "Interface2" 

component App {
  port port1
  port port2
}

Interface1 - port1
Interface2 - port2

port1 -> [Component 1]
port2 <- [Component 2]

@enduml
```
