package main

import (
	"fmt"

	"github.com/NathanBaulch/rainbow-roads/worms"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	wormsOpts = &worms.Options{
		Title:   Title,
		Version: Version,
	}
	wormsCmd = &cobra.Command{
		Use:   "worms",
		Short: "Animate exercise activities",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if wormsOpts.Frames == 0 {
				return flagError("frames", wormsOpts.Frames, "must be positive")
			}
			if wormsOpts.FPS == 0 {
				return flagError("fps", wormsOpts.FPS, "must be positive")
			}
			if wormsOpts.Width == 0 {
				return flagError("width", wormsOpts.Width, "must be positive")
			}
			if wormsOpts.ColorDepth == 0 {
				return flagError("color_depth", wormsOpts.ColorDepth, "must be positive")
			}
			if wormsOpts.Speed < 1 {
				return flagError("speed", wormsOpts.Speed, "must be greater than or equal to 1")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			wormsOpts.Input = args
			return worms.Run(wormsOpts)
		},
	}
)

func init() {
	rootCmd.AddCommand(wormsCmd)

	general := &pflag.FlagSet{}
	general.StringVarP(&wormsOpts.Output, "output", "o", "out", "optional path of the generated file")
	general.StringVarP(&wormsOpts.Format, "format", "f", "gif", "output file format string, supports gif, png, zip")
	general.VisitAll(func(f *pflag.Flag) { wormsCmd.Flags().Var(f.Value, f.Name, f.Usage) })

	rendering := &pflag.FlagSet{}
	rendering.UintVar(&wormsOpts.Frames, "frames", 200, "number of animation frames")
	rendering.UintVar(&wormsOpts.FPS, "fps", 20, "animation frame rate")
	rendering.UintVarP(&wormsOpts.Width, "width", "w", 500, "width of the generated image in pixels")
	_ = wormsOpts.Colors.Parse("#fff,#ff8,#911,#414,#007@.5,#003")
	rendering.Var((*ColorsFlag)(&wormsOpts.Colors), "colors", "CSS linear-colors inspired color scheme string, eg red,yellow,green,blue,black")
	rendering.UintVar(&wormsOpts.ColorDepth, "color_depth", 5, "number of bits per color in the image palette")
	rendering.Float64Var(&wormsOpts.Speed, "speed", 1.25, "how quickly activities should progress")
	rendering.BoolVar(&wormsOpts.Loop, "loop", false, "start each activity sequentially and animate continuously")
	rendering.BoolVar(&wormsOpts.NoWatermark, "no_watermark", false, "suppress the embedded project name and version string")
	rendering.VisitAll(func(f *pflag.Flag) { wormsCmd.Flags().Var(f.Value, f.Name, f.Usage) })

	filters := filterFlagSet(&wormsOpts.Selector)
	filters.VisitAll(func(f *pflag.Flag) { wormsCmd.Flags().Var(f.Value, f.Name, f.Usage) })

	wormsCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Fprintln(wormsCmd.OutOrStderr())
		fmt.Fprintln(wormsCmd.OutOrStderr(), "Usage:")
		fmt.Fprintln(wormsCmd.OutOrStderr(), " ", wormsCmd.UseLine(), "[input]")
		fmt.Fprintln(wormsCmd.OutOrStderr())
		fmt.Fprintln(wormsCmd.OutOrStderr(), "General flags:")
		fmt.Fprintln(wormsCmd.OutOrStderr(), general.FlagUsages())
		fmt.Fprintln(wormsCmd.OutOrStderr(), "Filtering flags:")
		fmt.Fprintln(wormsCmd.OutOrStderr(), filters.FlagUsages())
		fmt.Fprintln(wormsCmd.OutOrStderr(), "Rendering flags:")
		fmt.Fprint(wormsCmd.OutOrStderr(), rendering.FlagUsages())
		return nil
	})
}
