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
import { formSchema, FormType } from "@/lib/types";

export default function ConnectionForm(props: {
  onSubmit: (values: FormType) => void;
}) {
  const form = useForm<FormType>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      server: "",
    },
  });


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
                <FormDescription>piongs server address.</FormDescription>
                <div className="grow" />
                <FormMessage />
              </div>
            </FormItem>
          )}
        />
        <div className="flex">
          <div className="grow" />
          <Button type="submit">Submit</Button>
        </div>
      </form>
    </Form>
  );
}
