import * as React from "react";
import * as SelectPrimitive from "@radix-ui/react-select";
import { Check } from "lucide-react";

import { cn } from "@/lib/utils";

interface ImageProps {
  src: string;
  alt: string;
}

type SelectItemWithImageProps = React.ComponentPropsWithoutRef<
  typeof SelectPrimitive.Item
> &
  ImageProps;

const SelectItemWithImage = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Item>,
  SelectItemWithImageProps
>(({ className, children, src, alt, ...props }, ref) => (
  <SelectPrimitive.Item
    ref={ref}
    className={cn(
      "relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-2 pr-8 text-sm outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
      className,
    )}
    {...props}
  >
    <span className="absolute right-7 flex h-3.5 w-3.5 items-center justify-center">
      <SelectPrimitive.ItemIndicator>
        <Check className="h-6 w-6" strokeWidth={3} />
      </SelectPrimitive.ItemIndicator>
    </span>
    <div className="w-8" />
    <img src={src} alt={alt} className="max-h-10 rounded-sm md:max-h-28" />
    <div className="grow" />
    <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
    <div className="w-8" />
  </SelectPrimitive.Item>
));
SelectItemWithImage.displayName = "SelectItemWithImage";

export default SelectItemWithImage;
