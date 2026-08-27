<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { RefreshCw } from "@lucide/vue";
import { toast } from "vue-sonner";
import { FieldType, type Field } from "@/lib/wails";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { errorMessage } from "@/lib/errors";

const props = defineProps<{
  field: Field;
  modelValue: string;
  fetchOptions?: () => Promise<string[]>;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

function setValue(value: string | number) {
  emit("update:modelValue", String(value));
}

const fieldType = computed(() => String(props.field.type));
const fetching = ref(false);
const fetched = ref<string[]>([]);

watch(
  () => props.field.key,
  () => {
    fetched.value = [];
  },
);

const showFetchedSelect = computed(() => fetched.value.length > 0);

const selectOptions = computed(() => {
  const names = [...fetched.value];
  if (props.modelValue && !names.includes(props.modelValue)) {
    names.unshift(props.modelValue);
  }
  return names;
});

async function onFetch() {
  if (!props.fetchOptions) {
    return;
  }
  fetching.value = true;
  try {
    const names = (await props.fetchOptions()) ?? [];
    fetched.value = names;
    if (names.length === 0) {
      toast.error("No databases found");
      return;
    }
    if (!props.modelValue && names.length === 1) {
      setValue(names[0]);
    }
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    fetching.value = false;
  }
}
</script>

<template>
  <div class="grid min-w-0 gap-1.5">
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

    <InputGroup v-else-if="field.fetchable && fetchOptions">
      <div v-if="showFetchedSelect" class="min-w-0 flex-1">
        <Select
          :model-value="modelValue"
          @update:model-value="(v) => setValue(String(v ?? ''))"
        >
          <SelectTrigger
            :id="field.key"
            class="h-8 w-full min-w-0 rounded-none border-0 bg-transparent shadow-none focus-visible:ring-0 dark:bg-transparent dark:hover:bg-transparent"
          >
            <SelectValue :placeholder="field.placeholder || field.label" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="name in selectOptions" :key="name" :value="name">
              {{ name }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <InputGroupInput
        v-else
        :id="field.key"
        :model-value="modelValue"
        :placeholder="field.placeholder"
        :required="field.required"
        autocomplete="off"
        @update:model-value="setValue"
      />
      <InputGroupAddon align="inline-end">
        <InputGroupButton :disabled="fetching" @click="onFetch">
          <RefreshCw :class="fetching ? 'animate-spin' : undefined" />
          {{ fetching ? "Fetching…" : "Fetch" }}
        </InputGroupButton>
      </InputGroupAddon>
    </InputGroup>

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
