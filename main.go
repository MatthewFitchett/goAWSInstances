package main
	
import (
	"fmt"
    "bytes"
    "net/url"
	"net/http"
    "html/template"
	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/ec2"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome, %!", r.URL.Path[1:])
}

var tmpl = `
<!DOCTYPE html>
<html lang="en">
    <link href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css" rel="stylesheet" integrity="sha256-7s5uDGW3AHqw6xtJmNNtr+OBRJUlgkNJEo78P4b0yRw= sha512-nNo+yCHEyn0smMxSswnf/OnX6/KwJuZTlNZBjauKhTK0c+zT+q5JOCx0UFhXQ6rJR9jg6Es8gPuD2uZcYDLqSw==" crossorigin="anonymous">
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/js/bootstrap.min.js" integrity="sha256-KXn5puMvxCw+dAYznun+drMdG1IFl3agK0p/pqT9KAo= sha512-2e8qq0ETcfWRI4HJBzQiA3UoyFk6tbNyG+qSaIBZLyW9Xf3sWZHN/lxe9fTh1U45DpPf07yj94KsUHHWe4Yk1A==" crossorigin="anonymous"></script>
<body>

<div class="panel panel-default">
  <div class="panel-heading">
    <h3 class="panel-title">Marketplace Lab AWS Instance list</h3>
  </div>
  <div class="panel-body">
    <table class="table table-striped table-condensed"><tr><th>InstanceId</th><th>Name</th><th>Private IP</th><th>Instance Type</th><th>Public IP</th></tr>
    {{range .}}
        <tr>
        <td>{{.ID}}</td>
        <td>{{.Name}}</td>
        <td>{{.IP}}</td>
        <td>{{.Type}}</td>
        <td>{{.ExternalIP}}</td>
        </tr>
    {{end}}
  </table>
  </div>
</div>
</body>
</html>
`

type Instance struct {
  ID            string
  Name          string
  IP            string
  Type          string
  ExternalIP    string  
}

func NewInstance(id string, name string, ip string, instancetype string, externalip string) *Instance {
    p := new(Instance)
    p.ID = id
    p.Name = name
    p.IP = ip
    p.Type = instancetype
    p.ExternalIP = externalip
    return p
}

func handlerListEC2Instances(w http.ResponseWriter, r *http.Request) {

    svc := ec2.New(session.New(), &aws.Config{Region: aws.String("eu-west-1")})

    params := &ec2.DescribeInstancesInput{
        Filters: []*ec2.Filter{
            &ec2.Filter{
                Name: aws.String("instance-state-name"),
                Values: []*string{
                    aws.String(instanceState),
                },
            },
        },
    }

    instanceState := r.FormValue("state")
    if instanceState == "" {
        instanceState = "running"
    }

    // Describe all instances whilst applying the defined filter
    // http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-DescribeInstancesInput
    

    // Call the DescribeInstances Operation
    resp, err := svc.DescribeInstances(params)
    if err != nil {
        panic(err)
    }

    fmt.Println(" ** Checking for [" + instanceState + "] AWS eu-west-1 instances **")
    
    // Create a slice of the Instance struct type
    instances := make([]Instance, len(resp.Reservations[0].Instances))    
    
   // Loop through the instances. They don't always have a name-tag so set it to None if we can't find anything.
    for idx := range resp.Reservations {
        for _, inst := range resp.Reservations[idx].Instances {

            // We need to see if the Name is one of the tags. It's not always
            // present and not required in Ec2.
            name := "None"
            for _, keys := range inst.Tags {
                if *keys.Key == "Name" {
                    name = url.QueryEscape(*keys.Value)
                }
            }
            
            publicip := "None"
            for _, keys := range inst.Tags {
                if *keys.Key == "PublicIpAddress" {
                    publicip = url.QueryEscape(*keys.Value)
                }
            }
            
            i := NewInstance(*inst.InstanceId, name, *inst.PrivateIpAddress, *inst.InstanceType, publicip)
           // i := NewInstance(inst.InstanceId, name, inst.PrivateIpAddress, inst.InstanceType, inst.PublicIpAddress)
            instances = append(instances, *i)
        }
    }
    
    // Create the defined template
    t,err := template.New("name").Parse(tmpl)
    if err != nil {
        fmt.Println(err)
        return
    }

    // Parse the template into a buffer variable, passing in the instance collection, then grab the string of it
    var doc bytes.Buffer 
    err2 := t.Execute(&doc, instances) 
   
    if err2 != nil {
        fmt.Println(err2)
        return
    }
    s := doc.String()
    
    // Output the string having been escaped by template.HTML
    w.Write([]byte(template.HTML(s)))
}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/instances", handlerListEC2Instances)
	http.ListenAndServe(":8080", nil)
}