package options

import cli "gopkg.in/urfave/cli.v1"

// Options contains common controller options
type Options struct {
	Debug                 bool
	Kubeconfig            string
	LoadBalancerNamespace string
	LoadBalancerName      string
	PodNamespace          string
	PodName               string
}

// AddFlags add flags to app
func (opts *Options) AddFlags(app *cli.App) {

	flags := []cli.Flag{
		cli.StringFlag{
			Name:        "kubeconfig",
			Usage:       "Path to a kube config. Only required if out-of-cluster.",
			Destination: &opts.Kubeconfig,
		},
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "run with debug mode",
			Destination: &opts.Debug,
		},
		cli.StringFlag{
			Name:        "loadbalancer-namespace",
			EnvVar:      "LOADBALANCER_NAMESPACE",
			Usage:       "specify loadbalancer resource namespace",
			Destination: &opts.LoadBalancerNamespace,
		},
		cli.StringFlag{
			Name:        "loadbalancer-name",
			EnvVar:      "LOADBALANCER_NAME",
			Usage:       "specify loadbalancer resource name",
			Destination: &opts.LoadBalancerName,
		},
		cli.StringFlag{
			Name:        "pod-namespace",
			EnvVar:      "POD_NAMESPACE",
			Usage:       "specify pod namespace",
			Destination: &opts.PodNamespace,
		},
		cli.StringFlag{
			Name:        "pod-name",
			EnvVar:      "POD_NAME",
			Usage:       "specify pod name",
			Destination: &opts.PodName,
		},
	}

	app.Flags = append(app.Flags, flags...)
}
