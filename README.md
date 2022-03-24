# gql-audit
`gql-audit` is a simple command line program that makes it easy to search large graphql codebases to find usage of fields and types.

## Installation
The latest version of `gql-audit` can be found in the [releases](https://github.com/cozmo/gql-audit/releases) for this repo. Simply download the relevant binary and put it in your path.

## Usage
```
Usage:
  gql-audit [OPTIONS] PATHs...

Application Options:
  -s, --schema-path Path to the graphql schema file (can be JSON from an introspection query OR an SDL file).
  -f, --field-path  Type and field path to search for. Should be in the form 'TypeName.fieldName'.
  PATHs             The .graphql files to search for usage in. 

Help Options:
  -h, --help         Show this help message
```

## Example
Given this schema file

`schema.graphql`
```graphql
type Author {
  id: String!
  name: String!
}
type Todo {
  id: String!
  description: String!
  author: Author!
}
input TodoInput {
  description: String!
}
type Query {
  Todo(id: String!): Todo!
  AllTodos: [Todo!]!
}
type Mutation {
  createTodo(todo: TodoInput!): Todo
}
```

And these 2 query files
`queries/list.graphql`
```graphql
query GetAllTodos {
  AllTodos {
    id
    description
    author {
      id
      name
    }
  }
}

query GetTodo($id: String!) {
  Todo(id: $id) {
    id
    author {
      id
    }
  }
}
```

`queries/create.graphql`
```graphql
mutation CreateTodo($description: String!) {
  createTodo(todo: { description: $description }) {
    id
    author {
      name
    }
  }
}
```

You could run the following command to get a list of all the uses of the `name` field on the `Author` type.

```bash
$ gql-audit --schema-path schema.graphql --field-path Author.name ./queries/*.graphql
./queries/create.graphql:
  mutation.createTodo.author.name
./queries/list.graphql:
  query.AllTodos.author.name
```

Or you could run the following command to get a list of all the uses of the `createTodo` mutation.

```bash
$ gql-audit --schema-path schema.graphql --field-path Mutation.createTodo ./queries/*.graphql
./queries/create.graphql:
  mutation.createTodo
```
