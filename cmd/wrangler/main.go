package main
import ("fmt";"log";"net/http";"os";"github.com/stockyard-dev/stockyard-wrangler/internal/server";"github.com/stockyard-dev/stockyard-wrangler/internal/store")
func main(){port:=os.Getenv("PORT");if port==""{port="9700"};dataDir:=os.Getenv("DATA_DIR");if dataDir==""{dataDir="./wrangler-data"}
db,err:=store.Open(dataDir);if err!=nil{log.Fatalf("wrangler: %v",err)};defer db.Close();srv:=server.New(db)
fmt.Printf("\n  Wrangler — Self-hosted ETL and data pipeline runner\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n\n",port,port)
log.Printf("wrangler: listening on :%s",port);log.Fatal(http.ListenAndServe(":"+port,srv))}
