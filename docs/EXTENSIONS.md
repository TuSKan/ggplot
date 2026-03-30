# Extensions

Creating Custom `Geom` logic interfaces evaluating generating layouts!

## Implementing a Geom

To map custom physical output behaviors (like `geom.HexBin`), formally generate a function struct onto `ast.Geom`.
It requires defining explicit required checking ensuring users map structures.

```go
package custom_geom

import "github.com/TuSKan/ggplot/internal/ast"

func MyHexBin() ast.Geom {
 return ast.Geom{
 Type: "hexbin",
 RequiredAesthetics: []string{"x", "y"},
 }
}
```

Inject this directly :
```go
plot.New(ds).AddLayer(custom_geom.MyHexBin(), aes.X("long"), aes.Y("lat"))
```
