import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "./ui/select";
import { CodecInfoType, formSchema, FormType } from "@/lib/types";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { GetGameInfos } from "@/lib/datafetch";
import SelectItemWithImage from "./select-custom";

export default function ConnectionForm(props: {
  defaultServer: string;
  defaultCodec: CodecInfoType;
  onSubmit: (values: FormType) => void;
  onFirstSubmit: (server: FormType) => void;
}) {
  const form = useForm<FormType>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      ...props.defaultCodec,
      server: props.defaultServer,
      game: undefined,
    },
  });

  const [currentStage, setCurrentStage] = useState(1); // 1 for the first stage, 2 for the second stage

  // Fetch games based on the server URL
  const gamesQuery = useQuery({
    queryKey: ["games", form.watch("server")],
    queryFn: () => GetGameInfos(form.watch("server")!),
    // Only fetch if the server is entered and we're on the second stage
    enabled: currentStage === 2 && form.watch("server") !== "",
  });

  const handleNextStage = () => {
    if (currentStage === 1 && form.watch("server")) {
      props.onFirstSubmit(form.getValues());
      setCurrentStage(2); // Move to the second stage if server URL is provided
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(props.onSubmit)} className="space-y-8">
        <FormField
          control={form.control}
          name="server"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Server</FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <div className="flex flex-row">
                <FormDescription>Enter the server URL.</FormDescription>
                <div className="grow" />
                <FormMessage />
              </div>
              <div className="text-xs">
                When connecting to local server, add{" "}
                <code className="mx-1 rounded-full bg-zinc-800 px-1 py-0.5">
                  {"http://<ip>:<port>"}
                </code>{" "}
                and{" "}
                <code className="mx-1 rounded-full bg-zinc-800 px-1 py-0.5">
                  {"ws://<ip>:<port>"}
                </code>{" "}
                to "Insecure origions treated as secure" flag in{" "}
                <a href="chrome://flags" className="underline">
                  chrome://flags
                </a>
                .
              </div>
            </FormItem>
          )}
        />

        <div className="flex flex-row flex-wrap gap-5 gap-y-2">
          <FormField
            control={form.control}
            name="codec"
            render={({ field }) => (
              <FormItem>
                <Select
                  onValueChange={field.onChange}
                  defaultValue={field.value}
                >
                  <FormControl>
                    <SelectTrigger className="h-8 w-28">
                      <SelectValue placeholder="select a codec"></SelectValue>
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectGroup>
                      <SelectLabel>Codec</SelectLabel>
                      <SelectItem value="h264_nvenc">H.264</SelectItem>
                      <SelectItem value="hevc_nvenc">H.265</SelectItem>
                      <SelectItem value="av1_nvenc">AV1</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="frame_rate"
            render={({ field }) => (
              <FormItem>
                <Select
                  onValueChange={(v) => {
                    // v is like 30FPS, 60FPS, etc. We need to extract the number
                    const num = parseInt(v.match(/\d+/)![0]);
                    field.onChange(num);
                  }}
                  defaultValue={`${field.value}FPS`}
                >
                  <FormControl>
                    <SelectTrigger className="h-8 w-28">
                      <SelectValue placeholder="select a codec"></SelectValue>
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectGroup>
                      <SelectLabel>Frame Rate</SelectLabel>
                      <SelectItem value="30FPS">30FPS</SelectItem>
                      <SelectItem value="60FPS">60FPS</SelectItem>
                      <SelectItem value="90FPS">90FPS</SelectItem>
                      <SelectItem value="120FPS">120FPS</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="initial_bitrate"
            render={({ field }) => (
              <FormItem>
                <FormControl>
                  <Input
                    className="h-8 w-28"
                    ref={field.ref}
                    value={field.value / 1_000_000}
                    onChange={(e) => {
                      if (e.target.value) {
                        field.onChange(parseInt(e.target.value) * 1_000_000);
                      }
                    }}
                    type="number"
                  />
                </FormControl>
                <FormDescription>Initial Rate Mbps.</FormDescription>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="max_bitrate"
            render={({ field }) => (
              <FormItem>
                <FormControl>
                  <Input
                    className="h-8 w-28"
                    ref={field.ref}
                    value={field.value / 1_000_000}
                    onChange={(e) => {
                      if (e.target.value) {
                        field.onChange(parseInt(e.target.value) * 1_000_000);
                      }
                    }}
                    type="number"
                  />
                </FormControl>
                <FormDescription>Max Rate Mbps.</FormDescription>
              </FormItem>
            )}
          />
        </div>

        <FormField
          control={form.control}
          name="game"
          render={
            currentStage === 2 && gamesQuery.data
              ? ({ field: { value, onChange } }) => {
                  return (
                    <FormItem>
                      <FormLabel>Choose Game</FormLabel>
                      <FormControl>
                        <Select
                          defaultValue={value ? value.game_id : ""}
                          value={value ? value.game_id : ""}
                          onValueChange={(value) => {
                            // find game info with game_id same as value
                            const game = gamesQuery.data.find(
                              (game) => game.game_id === value,
                            );
                            if (game) {
                              onChange(game);
                            }
                          }}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="Select a game" />
                          </SelectTrigger>
                          <SelectContent>
                            {gamesQuery.data.map((game) => (
                              <SelectItemWithImage
                                key={game.game_id}
                                value={game.game_id}
                                src={`https://shared.cloudflare.steamstatic.com/store_item_assets/steam/apps/${game.game_id}/header.jpg`}
                                alt=""
                                className="text-md md:text-2xl"
                              >
                                {game.game_display_name}
                              </SelectItemWithImage>
                            ))}
                          </SelectContent>
                        </Select>
                      </FormControl>
                      <div className="flex flex-row">
                        <FormDescription>
                          Select a game from the list.
                        </FormDescription>
                        <div className="grow" />
                        <FormMessage />
                      </div>
                    </FormItem>
                  );
                }
              : () => <></>
          }
        />

        <div className="flex">
          <div className="grow" />
          <Button
            type={currentStage === 1 ? "button" : "submit"}
            onClick={currentStage === 1 ? handleNextStage : undefined}
            disabled={
              gamesQuery.isLoading ||
              (currentStage === 1 && !form.watch("server"))
            }
          >
            {currentStage === 1 ? "Next" : "Connect"}
          </Button>
        </div>
      </form>
    </Form>
  );
}
