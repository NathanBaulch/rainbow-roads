package main

import (
	"fmt"

	"github.com/NathanBaulch/rainbow-roads/paint"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	paintOpts = &paint.Options{
		Title:   Title,
		Version: Version,
	}
	paintCmd = &cobra.Command{
		Use:   "paint",
		Short: "Track coverage in a region of interest",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if paintOpts.Width == 0 {
				return flagError("width", paintOpts.Width, "must be positive")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			paintOpts.Input = args
			return paint.Run(paintOpts)
		},
	}
)

func init() {
	rootCmd.AddCommand(paintCmd)

	general := &pflag.FlagSet{}
	general.VarP((*CircleFlag)(&paintOpts.Region), "region", "r", "target region of interest, eg -37.8,144.9,10km")
	general.StringVarP(&paintOpts.Output, "output", "o", "out", "optional path of the generated file")
	general.VisitAll(func(f *pflag.Flag) { paintCmd.Flags().Var(f.Value, f.Name, f.Usage) })
	_ = paintCmd.MarkFlagRequired("region")

	rendering := &pflag.FlagSet{}
	rendering.UintVarP(&paintOpts.Width, "width", "w", 1000, "width of the generated image in pixels")
	rendering.BoolVar(&paintOpts.NoWatermark, "no_watermark", false, "suppress the embedded project name and version string")
	rendering.VisitAll(func(f *pflag.Flag) { paintCmd.Flags().Var(f.Value, f.Name, f.Usage) })

	filters := filterFlagSet(&paintOpts.Selector)
	filters.VisitAll(func(f *pflag.Flag) { paintCmd.Flags().Var(f.Value, f.Name, f.Usage) })

	paintCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Fprintln(paintCmd.OutOrStderr())
		fmt.Fprintln(paintCmd.OutOrStderr(), "Usage:")
		fmt.Fprintln(paintCmd.OutOrStderr(), " ", paintCmd.UseLine(), "[input]")
		fmt.Fprintln(paintCmd.OutOrStderr())
		fmt.Fprintln(paintCmd.OutOrStderr(), "General flags:")
		fmt.Fprintln(paintCmd.OutOrStderr(), general.FlagUsages())
		fmt.Fprintln(paintCmd.OutOrStderr(), "Filtering flags:")
		fmt.Fprintln(paintCmd.OutOrStderr(), filters.FlagUsages())
		fmt.Fprintln(paintCmd.OutOrStderr(), "Rendering flags:")
		fmt.Fprint(paintCmd.OutOrStderr(), rendering.FlagUsages())
		return nil
	})
}
