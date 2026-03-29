# Extensions

Creating Custom `Geom` logic seamlessly interfaces evaluating bounds securely generating layouts securely mapping correctly dynamically structurally!

## Implementing a Geom

To map custom physical output behaviors cleanly securely (like `geom.HexBin`), formally generate a function struct mapped onto `ast.Geom`.
It requires defining explicit required mappings natively checking explicitly smoothly natively ensuring users map structures natively properly.

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

Inject this directly mapping correctly:
```go
plot.New(ds).AddLayer(custom_geom.MyHexBin(), aes.X("long"), aes.Y("lat"))
```
