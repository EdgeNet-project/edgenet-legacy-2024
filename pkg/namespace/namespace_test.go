package namespace
import (
	"testing"
	"fmt"
	testclient "k8s.io/client-go/kubernetes/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"	
)

func TestCreate(t *testing.T) {
	cases := []struct {
		ns string
		
	} {
			 {"test"},
			 {"test1"},		
	}

	for _, c := range cases {
		client := testclient.NewSimpleClientset()
		result, err := Create(c.ns, client)
		fmt.Println(GetList(client))
		if err != nil {
			t.Fatal(err.Error())
		}
		
		if result != c.ns {
			t.Fatal("result different from namespace")
		}	
	}
}

func TestDelete(t *testing.T){
	cases := []struct {
		ns string
	}{
		{"test"},
		{"test1"},
		{"test2"},
	}

	for _, c := range cases {
	   	client := testclient.NewSimpleClientset()
		result, _ := Create(c.ns, client)
		resultD, err := Delete(result, client)
		fmt.Println(resultD);
		if err != nil{
			t.Fatal(err)
		}
		if (resultD != "deleted" &&  resultD != "" ){
			t.Fatal("not deleted")

		}
	}
} 


func TestGetList(t *testing.T){
	client := testclient.NewSimpleClientset()
	cases := []struct {
		ns string
	}{
		{"test"},
		{"test1"},
		{"test2"},
	}
	var resultat [] string

	for _, c := range cases {
		result, _ := Create(c.ns, client)
		resultat = append(resultat, result)	
	}
	for index,c := range GetList(client) {
		if c != resultat[index]{
			t.Fatal("Error!!!")
		}	
	}
}


func TestGetNamespaceByName(t *testing.T) {
  data := []struct {
  		clientset      kubernetes.Interface
  		ns             string
     		expected       string
  	}{
  		{  clientset: testclient.NewSimpleClientset(&corev1.Namespace{
  				ObjectMeta: metav1.ObjectMeta{
  					Name:        "namespace1",
  					Namespace:   "default", },
  			}, &corev1.Namespace{
  				ObjectMeta: metav1.ObjectMeta{
  					Name:        "namespace2",
  					Namespace:   "default", },
  			}),
        		ns: "namespace1",
        		expected :"true", },
  		{   clientset: testclient.NewSimpleClientset(&corev1.Namespace{
  				ObjectMeta: metav1.ObjectMeta{
  					Name:        "namespace3",
  					Namespace:   "default", },
  			}, &corev1.Namespace{
  				ObjectMeta: metav1.ObjectMeta{
  					Name:        "namespace4",
  					Namespace:   "default", },
  			}),
       			 ns: "namespace3",
        		expected: "true",
		}, }
  for _, test := range data {
    if output, err := GetNamespaceByName( test.ns, test.clientset); output != test.expected {
			t.Error(err)
		}
	}
}
