<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import DynamicField from "@/features/connections/DynamicField.vue";
import { errorMessage } from "@/lib/errors";
import type { Field, Profile, SaveProfileInput } from "@/lib/wails";
import { useConnectionsStore } from "@/stores/connections";

const props = defineProps<{
  open: boolean;
  profile?: Profile | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
}>();

const connections = useConnectionsStore();
const name = ref("");
const driverId = ref("");
const params = reactive<Record<string, string>>({});
const saving = ref(false);
const testing = ref(false);

const spec = computed(() => connections.specFor(driverId.value));
const isEdit = computed(() => Boolean(props.profile?.id));

type FieldRow = { id: string; fields: Field[] };

const fieldRows = computed((): FieldRow[] => {
  const rows: FieldRow[] = [];
  const groupIndex = new Map<string, number>();
  for (const field of spec.value) {
    const group = field.group;
    if (!group) {
      rows.push({ id: field.key, fields: [field] });
      continue;
    }
    const existing = groupIndex.get(group);
    if (existing === undefined) {
      groupIndex.set(group, rows.length);
      rows.push({ id: group, fields: [field] });
    } else {
      rows[existing].fields.push(field);
    }
  }
  return rows;
});

function applyDriver(nextDriverId: string, existing?: Record<string, string>) {
  driverId.value = nextDriverId;
  const nextSpec = connections.specFor(nextDriverId);
  const keys = new Set(nextSpec.map((field) => field.key));
  for (const key of Object.keys(params)) {
    if (!keys.has(key)) {
      delete params[key];
    }
  }
  for (const field of nextSpec) {
    if (field.secret) {
      params[field.key] = "";
      continue;
    }
    params[field.key] = existing?.[field.key] ?? field.default ?? "";
  }
}

watch(
  () => [props.open, props.profile] as const,
  () => {
    if (!props.open) {
      return;
    }
    const fallbackDriver = connections.drivers[0]?.id ?? "";
    name.value = props.profile?.name ?? "";
    applyDriver(props.profile?.driverId || fallbackDriver, connections.asRecord(props.profile?.params ?? null));
  },
  { immediate: true },
);

function onDriverChange(value: unknown) {
  applyDriver(String(value ?? ""), params);
}

function payload(): SaveProfileInput {
  return {
    id: props.profile?.id ?? "",
    name: name.value.trim(),
    driverId: driverId.value,
    params: { ...params },
  };
}

async function onTest() {
  testing.value = true;
  try {
    await connections.test(payload());
    toast.success("Connection succeeded");
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    testing.value = false;
  }
}

async function onSave() {
  saving.value = true;
  try {
    await connections.save(payload());
    toast.success(isEdit.value ? "Connection saved" : "Connection created");
    emit("update:open", false);
  } catch (err) {
    toast.error(errorMessage(err));
  } finally {
    saving.value = false;
  }
}

async function fetchDatabases() {
  return connections.fetchDatabases(payload());
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent v-if="open" class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ isEdit ? "Edit connection" : "New connection" }}</DialogTitle>
        <DialogDescription>
          Driver fields come from the backend spec, so new engines do not need a custom form.
        </DialogDescription>
      </DialogHeader>

      <form class="grid gap-3" @submit.prevent="onSave">
        <div class="grid gap-1.5">
          <Label for="connection-name">Name</Label>
          <Input id="connection-name" v-model="name" placeholder="Production" required />
        </div>

        <div class="grid gap-1.5">
          <Label>Driver</Label>
          <Select :model-value="driverId" @update:model-value="onDriverChange">
            <SelectTrigger class="w-full">
              <SelectValue placeholder="Select a driver" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="driver in connections.drivers" :key="driver.id" :value="driver.id">
                {{ driver.displayName }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div
          v-for="row in fieldRows"
          :key="row.id"
          :class="row.fields.length > 1 ? 'grid grid-cols-2 gap-2' : undefined"
        >
          <DynamicField
            v-for="field in row.fields"
            :key="`${driverId}:${field.key}`"
            :field="field"
            :model-value="params[field.key] ?? ''"
            :fetch-options="field.fetchable ? fetchDatabases : undefined"
            @update:model-value="params[field.key] = $event"
          />
        </div>

        <DialogFooter class="mt-2">
          <Button type="button" variant="outline" :disabled="testing || saving" @click="onTest">
            {{ testing ? "Testing…" : "Test" }}
          </Button>
          <Button type="submit" :disabled="saving || testing">
            {{ saving ? "Saving…" : "Save" }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
