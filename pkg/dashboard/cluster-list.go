package dashboard

import (
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"k8s.io/klog"
	"net/http"
)

// NamespaceList replies with the rendered namespace list of all goldilocks enabled namespaces
func ClusterList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read entire file content, giving us little control but
		// making it very simple. No need to close the file.
		//content, err := ioutil.ReadFile(*kubeconfig)
		//if err != nil {
		//    log.Fatal(err)
		//}
		//
		//kubeConfigData, err := clientcmd.Load(content)
		//if err != nil {
		//   panic(err.Error())
		//}
		//for _, c := range kubeConfigData.Contexts {
		//    klog.Info("Context found: %s", c)
		//}
		//// use the current context in kubeconfig
		//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		//if err != nil {
		//    panic(err.Error())
		//}

		contexts := kube.GetContexts(opts.kubeconfigPath)
		for x, y := range contexts {
			klog.Infof("cluster name: %s, context %s", x, y)
		}

		//// create the clientset
		//clientset, err := kubernetes.NewForConfig(config)
		//if err != nil {
		//   panic(err.Error())
		//}
		//klog.Info("clientset infos: %v", clientset)

		//namespacesList, err := kube.GetInstance().Client.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{
		//	LabelSelector: labels.Set(map[string]string{
		//		utils.VpaEnabledLabel: "true",
		//	}).String(),
		//})
		//if err != nil {
		//	klog.Errorf("Error getting namespace list: %v", err)
		//	http.Error(w, "Error getting namespace list", http.StatusInternalServerError)
		//}

		//tmpl, err := getTemplate("namespace_list",
		//	"namespace_list",
		//)
		//if err != nil {
		//	klog.Errorf("Error getting template data: %v", err)
		//	http.Error(w, "Error getting template data", http.StatusInternalServerError)
		//	return
		//}

		// only expose the needed data from Namespace
		// this helps to not leak additional information like
		// annotations, labels, metadata about the Namespace to the
		// client UI source code or javascript console
		//data := []struct {
		//	Name string
		//}{}

		//for _, ns := range namespacesList.Items {
		//	item := struct {
		//		Name string
		//	}{
		//		Name: ns.Name,
		//	}
		//	data = append(data, item)
		//}

		//writeTemplate(tmpl, opts, &data, w)
	})
}
