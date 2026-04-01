import {
  IconCheck,
  IconLoader2,
  IconX,
} from "@tabler/icons-react"
import { useState, useEffect } from "react"

import type { ConfigField, MarketplaceSkill } from "./hooks/useMarketplace"
import { useSkillConfig } from "./hooks/useMarketplace"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Separator } from "@/components/ui/separator"

interface SkillConfigurationProps {
  skill: MarketplaceSkill
}

export function SkillConfiguration({ skill }: SkillConfigurationProps) {
  const { saveConfig, isSaving, testConnection, isTesting, testResult, clearTestResult } =
    useSkillConfig(skill.id)

  const [config, setConfig] = useState<Record<string, string | number | boolean>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (skill.configSchema) {
      const initialConfig: Record<string, string | number | boolean> = {}
      skill.configSchema.forEach((field) => {
        initialConfig[field.name] = field.default ?? (field.type === "boolean" ? false : "")
      })
      setConfig(initialConfig)
    }
  }, [skill.configSchema])

  const handleFieldChange = (name: string, value: string | number | boolean) => {
    setConfig((prev) => ({ ...prev, [name]: value }))
    setErrors((prev) => {
      const next = { ...prev }
      delete next[name]
      return next
    })
    clearTestResult()
  }

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {}
    skill.configSchema?.forEach((field) => {
      if (field.required) {
        const value = config[field.name]
        if (value === "" || value === undefined || value === null) {
          newErrors[field.name] = `${field.label} is required`
        }
      }
    })
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleTest = () => {
    if (validate()) {
      testConnection(config)
    }
  }

  const handleSave = () => {
    if (validate()) {
      saveConfig({ config })
    }
  }

  const renderField = (field: ConfigField) => {
    const value = config[field.name]
    const error = errors[field.name]

    switch (field.type) {
      case "text":
      case "password":
        return (
          <div key={field.name} className="space-y-2">
            <Label htmlFor={field.name} className="text-sm font-medium">
              {field.label}
              {field.required && <span className="text-destructive ml-0.5">*</span>}
            </Label>
            {field.description && (
              <p className="text-muted-foreground text-xs">{field.description}</p>
            )}
            <Input
              id={field.name}
              type={field.type}
              value={(value as string) ?? ""}
              onChange={(e) => handleFieldChange(field.name, e.target.value)}
              placeholder={`Enter ${field.label.toLowerCase()}`}
              className={error ? "border-destructive" : ""}
            />
            {error && <p className="text-destructive text-xs">{error}</p>}
          </div>
        )

      case "number":
        return (
          <div key={field.name} className="space-y-2">
            <Label htmlFor={field.name} className="text-sm font-medium">
              {field.label}
              {field.required && <span className="text-destructive ml-0.5">*</span>}
            </Label>
            {field.description && (
              <p className="text-muted-foreground text-xs">{field.description}</p>
            )}
            <Input
              id={field.name}
              type="number"
              value={(value as number) ?? ""}
              onChange={(e) => handleFieldChange(field.name, Number(e.target.value))}
              placeholder={`Enter ${field.label.toLowerCase()}`}
              className={error ? "border-destructive" : ""}
            />
            {error && <p className="text-destructive text-xs">{error}</p>}
          </div>
        )

      case "boolean":
        return (
          <div key={field.name} className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor={field.name} className="text-sm font-medium">
                {field.label}
              </Label>
              {field.description && (
                <p className="text-muted-foreground text-xs">{field.description}</p>
              )}
            </div>
            <Switch
              id={field.name}
              checked={(value as boolean) ?? false}
              onCheckedChange={(checked) => handleFieldChange(field.name, checked)}
              size="sm"
            />
          </div>
        )

      case "select":
        return (
          <div key={field.name} className="space-y-2">
            <Label htmlFor={field.name} className="text-sm font-medium">
              {field.label}
              {field.required && <span className="text-destructive ml-0.5">*</span>}
            </Label>
            {field.description && (
              <p className="text-muted-foreground text-xs">{field.description}</p>
            )}
            <Select
              value={(value as string) ?? ""}
              onValueChange={(v) => handleFieldChange(field.name, v)}
            >
              <SelectTrigger id={field.name} className={error ? "border-destructive" : ""}>
                <SelectValue placeholder={`Select ${field.label.toLowerCase()}`} />
              </SelectTrigger>
              <SelectContent>
                {field.options?.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {error && <p className="text-destructive text-xs">{error}</p>}
          </div>
        )

      default:
        return null
    }
  }

  if (!skill.configSchema || skill.configSchema.length === 0) {
    return (
      <div className="text-muted-foreground py-8 text-center text-sm">
        No configuration options available for this skill.
      </div>
    )
  }

  return (
    <div className="space-y-6 py-6">
      <div className="space-y-4">
        {skill.configSchema.map((field) => renderField(field))}
      </div>

      {testResult && (
        <div
          className={`flex items-center gap-2 rounded-md p-3 text-sm ${
            testResult.success
              ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
              : "bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"
          }`}
        >
          {testResult.success ? (
            <IconCheck className="size-4" />
          ) : (
            <IconX className="size-4" />
          )}
          {testResult.message || (testResult.success ? "Connection successful" : "Connection failed")}
        </div>
      )}

      <Separator />

      <div className="flex items-center gap-2">
        <Button variant="outline" onClick={handleTest} disabled={isTesting}>
          {isTesting ? (
            <IconLoader2 className="size-4 animate-spin" />
          ) : null}
          Test Connection
        </Button>
        <Button onClick={handleSave} disabled={isSaving}>
          {isSaving ? (
            <IconLoader2 className="size-4 animate-spin" />
          ) : null}
          Save Configuration
        </Button>
      </div>
    </div>
  )
}

export default SkillConfiguration
