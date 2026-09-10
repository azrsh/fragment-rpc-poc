@attached(member, names: named(Data), named(document))
public macro GraphQLFragment(_ document: String) = #externalMacro(module: "GraphQLMacros", type: "GraphQLMacro")

@attached(member, names: named(Data), named(document))
public macro GraphQLQuery(_ document: String) = #externalMacro(module: "GraphQLMacros", type: "GraphQLMacro")
