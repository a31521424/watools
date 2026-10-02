package secrets
import("strings";"testing")
func TestNamespaceValidation(t *testing.T){
 for _,id:=range []string{"", "../escape", "watools.plugin.test/other"}{if _,err:=account(id,"key");err==nil{t.Fatalf("unsafe package accepted: %s",id)}}
 a,err:=account("watools.plugin.first","key");if err!=nil{t.Fatal(err)};b,err:=account("watools.plugin.second","key");if err!=nil{t.Fatal(err)}
 if a==b || len(a)!=64 || strings.Contains(a,"key"){t.Fatal("secret namespaces collide")}
}
