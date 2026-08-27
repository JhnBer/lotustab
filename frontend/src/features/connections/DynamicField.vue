<script setup lang="ts">
import { computed } from "vue";
import { FieldType, type Field } from "@/lib/wails";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const props = defineProps<{
  field: Field;
  modelValue: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

function setValue(value: string | number) {
  emit("update:modelValue", String(value));
}

const fieldType = computed(() => String(props.field.type));
</script>

<template>
  <div class="grid gap-1.5">
    <Label :for="field.key">{{ field.label }}</Label>

    <Select
      v-if="fieldType === FieldType.FieldSelect || fieldType === 'select'"
      :model-value="modelValue || field.default || ''"
      @update:model-value="(v) => setValue(String(v ?? ''))"
    >
      <SelectTrigger class="w-full">
        <SelectValue :placeholder="field.placeholder || field.label" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem
          v-for="option in field.options ?? []"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </SelectItem>
      </SelectContent>
    </Select>

    <div
      v-else-if="fieldType === FieldType.FieldCheckbox || fieldType === 'checkbox'"
      class="flex items-center gap-2 py-1"
    >
      <Checkbox
        :id="field.key"
        :checked="modelValue === 'true'"
        @update:checked="(checked) => setValue(checked ? 'true' : 'false')"
      />
      <Label :for="field.key" class="font-normal">{{ field.label }}</Label>
    </div>

    <Input
      v-else
      :id="field.key"
      :model-value="modelValue"
      :type="fieldType === 'password' || field.secret ? 'password' : fieldType === 'number' ? 'number' : 'text'"
      :placeholder="field.placeholder"
      :required="field.required"
      autocomplete="off"
      @update:model-value="setValue"
    />
  </div>
</template>
