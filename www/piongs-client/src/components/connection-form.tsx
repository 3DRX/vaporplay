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
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formSchema, FormType } from "@/lib/types";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { GetGameInfos } from "@/lib/datafetch";

export default function ConnectionForm(props: {
  onSubmit: (values: FormType) => void;
}) {
  const form = useForm<FormType>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      server: "",
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
            </FormItem>
          )}
        />

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
                              <SelectItem
                                key={game.game_id}
                                value={game.game_id}
                              >
                                {game.game_display_name}
                              </SelectItem>
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
