package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    
    "github.com/gorilla/mux"
    "github.com/graphql-go/graphql"
)

// Criando um cadastro de URLs, logo temos que ter o tipo URL
type Url struct {
    Name string `json:"name"`  // Não tem vírgula aqui!
    SiteUrl string `json:"siteurl"`
}

// Para a extração da query do corpo da requisição:
// {"query": "..."}
type PostData struct {
    Query string `json:"query"`
}

// Nosso "banco de dados"" já inicialmente populado
var UrlList = []Url{  // conceito de slice aqui!
    Url{Name: "google", SiteUrl: "google.com"},
}

// Schema no motor de GraphQL
var UrlType = graphql.NewObject(graphql.ObjectConfig{
    Name: "Url",
    Fields: graphql.Fields{
        // Atenção a onde é o struct e onde é ponteiro
        "name": &graphql.Field{
            Type: graphql.String,
        },
        "siteurl": &graphql.Field{
            Type: graphql.String,
        },
    },
})

// Queries: consultar 1 URL e uma lista de URLs
var RootQuery = graphql.NewObject(graphql.ObjectConfig{
    Name: "RootQuery",
    Fields: graphql.Fields{
        // {url(name:\"google\") {name siteurl}}
        "url": &graphql.Field{
            Type: UrlType,  // tipo do retorno
            Description: "Obter uma única URL",
            // que APIzinha sem-vergonha de javeira :P
            Args: graphql.FieldConfigArgument{
                // Olha o ponteiro: como você cria sem uma referência pronta?
                "name": &graphql.ArgumentConfig{
                    Type: graphql.String,
                },
            },
            Resolve: GetResolver,
        },
        
        // {urllist {name siteurl}}
        "urllist": &graphql.Field{
            Type: graphql.NewList(UrlType),  // retorna lista
            Description: "Lista de URLs",
            Resolve: GetListResolver,
        },
    },
})

// Mutações: adicionar e remover URL
var RootMutation = graphql.NewObject(graphql.ObjectConfig{
    Name: "RootMutation",
    Fields: graphql.Fields{
        // mutation { createUrl(name:\"youtube\", siteurl:\"youtube.com\") { name siteurl } }
        "createUrl": &graphql.Field{
            Type: UrlType,
            Description: "Criar nova URL",
            Args: graphql.FieldConfigArgument{
                "name": &graphql.ArgumentConfig{
                    // Forçando strings não nulas!
                    Type: graphql.NewNonNull(graphql.String),
                },
                "siteurl": &graphql.ArgumentConfig{
                    Type: graphql.NewNonNull(graphql.String),
                },
            },
            Resolve: CreateNewUrlResolver,
        },
        
        // mutation { deleteUrl(name:\"youtube\") { name siteurl } }
        "deleteUrl": &graphql.Field{
            Type: UrlType,
            Description: "Apagar uma URL",
            Args: graphql.FieldConfigArgument{
                "name": &graphql.ArgumentConfig{
                    Type: graphql.String,
                },
            },
            Resolve: DeleteUrlResolver,
        },
    },
})

// Enfim, o grande schema do GraphQL!
var UrlSchema, _ = graphql.NewSchema(graphql.SchemaConfig{
    Query: RootQuery,
    Mutation: RootMutation,
})

// Os resolvedores de problemas, ops, de queries :)
// Todos eles têm a mesma assinatura exigida pela lib.
var GetResolver = func(params graphql.ResolveParams) (interface{}, error) {
    // ponto + parênteses: forçando tipo
    idQuery, isOk := params.Args["name"].(string)
    
    if isOk {
        // Ainda se lembra como percorre lista?
        for _, url := range UrlList {
            if url.Name == idQuery {
                return url, nil
            }
        }
    }
    
    return Url{}, nil
}

var GetListResolver = func(p graphql.ResolveParams) (interface{}, error) {
    return UrlList, nil
}

var CreateNewUrlResolver = func(params graphql.ResolveParams) (interface{}, error) {
    // Ignorando os erros, qual o efeito?
    // Foi definido que os tipos destes parâmetros é graphql.NewNonNull(graphql.String)
    nameProp, _ := params.Args["name"].(string)
    siteurlProp, _ := params.Args["siteurl"].(string)
    newUrl := Url{Name: nameProp, SiteUrl: siteurlProp}
    
    // Novo banco NoSQL irá hypar!
    UrlList = append(UrlList, newUrl)
    
    return newUrl, nil
}

var DeleteUrlResolver = func(params graphql.ResolveParams) (interface{}, error) {
    nameUrl, _ := params.Args["name"].(string)
    deletedUrl := Url{}
    
    for i := 0; i < len(UrlList); i++ {
        if UrlList[i].Name == nameUrl {
            deletedUrl = UrlList[i]
            // A elegância de como é remover um item da lista...
            UrlList = append(UrlList[:i], UrlList[i+1:]...)
            break
        }
    }
    
    return deletedUrl, nil
}

// Agora o tratador de HTTP!
func ProcessGraphQL(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    defer r.Body.Close()
    var query PostData
    
    if err := decoder.Decode(&query); err != nil {
        WriteResponse(http.StatusBadRequest, map[string]string{"error": err.Error()}, w)
        return
    }
    
    // Isto executa o GraphQL!
    result := graphql.Do(graphql.Params{
        Context: r.Context(),
        Schema: UrlSchema,  // o big schema acima
        RequestString: query.Query,  // lemos o campo "query" do JSON da requisição
    })
    
    WriteResponse(http.StatusOK, result, w)
}

func WriteResponse(status int, body interface{}, w http.ResponseWriter) {
    w.WriteHeader(status)
    w.Header().Set("Content-Type", "application/json")
    payload, _ := json.Marshal(body)
    w.Write(payload)
}

func main() {
    router := mux.NewRouter()
    router.HandleFunc("/graphql", ProcessGraphQL).Methods("POST")
    err := http.ListenAndServe("localhost:8080", router)
    fmt.Println(err)
}

