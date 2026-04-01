import {
  IconBook,
  IconCalendar,
  IconCheck,
  IconDownload,
  IconExternalLink,
  IconLoader2,
  IconStar,
  IconStarFilled,
  IconTrash,
  IconUpload,
} from "@tabler/icons-react"
import { useState } from "react"
import ReactMarkdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import type { MarketplaceSkill, SkillReview } from "./hooks/useMarketplace"
import { useSkillReviews } from "./hooks/useMarketplace"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Separator } from "@/components/ui/separator"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"

interface SkillDetailSheetProps {
  skill: MarketplaceSkill | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onInstall: (skillId: string) => void
  onUninstall: (skillId: string) => void
  onUpdate: (skillId: string) => void
  onSubmitRating: (skillId: string, rating: number, comment?: string) => void
  isInstalling?: boolean
  isUninstalling?: boolean
  isUpdating?: boolean
  isSubmittingRating?: boolean
}

export function SkillDetailSheet({
  skill,
  open,
  onOpenChange,
  onInstall,
  onUninstall,
  onUpdate,
  onSubmitRating,
  isInstalling,
  isUninstalling,
  isUpdating,
  isSubmittingRating,
}: SkillDetailSheetProps) {
  const [userRating, setUserRating] = useState(0)
  const [reviewComment, setReviewComment] = useState("")
  const [activeTab, setActiveTab] = useState<"details" | "reviews">("details")

  const { reviews, isLoading: isLoadingReviews } = useSkillReviews(skill?.id ?? "")

  if (!skill) return null

  const formatPrice = (price: number): string => {
    return price === 0 ? "Free" : `$${price.toFixed(2)}`
  }

  const formatDate = (dateStr: string): string => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    })
  }

  const handleSubmitRating = () => {
    if (userRating > 0) {
      onSubmitRating(skill.id, userRating, reviewComment || undefined)
      setUserRating(0)
      setReviewComment("")
    }
  }

  const renderStars = (rating: number, interactive = false, onClick?: (rating: number) => void) => {
    const stars = []
    for (let i = 1; i <= 5; i++) {
      const filled = i <= rating
      stars.push(
        interactive ? (
          <button
            key={i}
            type="button"
            onClick={() => onClick?.(i)}
            className="cursor-pointer p-0.5 transition-transform hover:scale-110"
          >
            {filled ? (
              <IconStarFilled className="size-5 text-yellow-500" />
            ) : (
              <IconStar className="size-5 text-muted-foreground/30" />
            )}
          </button>
        ) : filled ? (
          <IconStarFilled key={i} className="size-4 text-yellow-500" />
        ) : (
          <IconStar key={i} className="size-4 text-muted-foreground/30" />
        )
      )
    }
    return stars
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-full gap-0 p-0 sm:!w-[600px] sm:!max-w-[600px]"
        showCloseButton
      >
        <SheetHeader className="border-b px-6 py-5">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <SheetTitle className="text-xl">{skill.name}</SheetTitle>
              <SheetDescription className="mt-1">
                by {skill.author} • v{skill.version}
              </SheetDescription>
            </div>
            <div className="text-right">
              <div className="text-lg font-semibold">{formatPrice(skill.price)}</div>
              <div className="mt-1 flex items-center gap-1">
                {renderStars(skill.rating)}
                <span className="text-muted-foreground ml-1 text-sm">
                  ({skill.rating.toFixed(1)})
                </span>
              </div>
            </div>
          </div>

          <div className="mt-4 flex gap-2">
            <Button
              size="sm"
              variant={activeTab === "details" ? "default" : "outline"}
              onClick={() => setActiveTab("details")}
            >
              Details
            </Button>
            <Button
              size="sm"
              variant={activeTab === "reviews" ? "default" : "outline"}
              onClick={() => setActiveTab("reviews")}
            >
              Reviews
            </Button>
          </div>
        </SheetHeader>

        <ScrollArea className="flex-1">
          {activeTab === "details" ? (
            <div className="space-y-6 p-6">
              <div>
                <h3 className="mb-2 text-sm font-medium">Description</h3>
                <div className="prose prose-sm dark:prose-invert max-w-none">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    rehypePlugins={[rehypeRaw, rehypeSanitize]}
                  >
                    {skill.longDescription || skill.description}
                  </ReactMarkdown>
                </div>
              </div>

              <div className="flex flex-wrap gap-2">
                <span className="bg-primary/10 text-primary rounded px-2 py-1 text-xs font-medium">
                  {skill.category}
                </span>
                {skill.tags.map((tag) => (
                  <span
                    key={tag}
                    className="bg-muted text-muted-foreground rounded px-2 py-1 text-xs font-medium"
                  >
                    {tag}
                  </span>
                ))}
              </div>

              {skill.screenshots && skill.screenshots.length > 0 && (
                <div>
                  <h3 className="mb-2 text-sm font-medium">Screenshots</h3>
                  <div className="grid grid-cols-2 gap-2">
                    {skill.screenshots.map((src, i) => (
                      <img
                        key={i}
                        src={src}
                        alt={`Screenshot ${i + 1}`}
                        className="rounded-lg border"
                      />
                    ))}
                  </div>
                </div>
              )}

              {skill.documentationUrl && (
                <div>
                  <a
                    href={skill.documentationUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary inline-flex items-center gap-1 text-sm hover:underline"
                  >
                    <IconBook className="size-4" />
                    Documentation
                    <IconExternalLink className="size-3" />
                  </a>
                </div>
              )}

              {skill.versionHistory && skill.versionHistory.length > 0 && (
                <div>
                  <h3 className="mb-2 text-sm font-medium">Version History</h3>
                  <div className="space-y-3">
                    {skill.versionHistory.slice(0, 5).map((v, i) => (
                      <div key={v.version} className="flex gap-3">
                        <div className="flex flex-col items-center">
                          <div className="bg-muted size-2 rounded-full" />
                          {i < skill.versionHistory!.length - 1 && (
                            <div className="bg-border w-px flex-1" />
                          )}
                        </div>
                        <div className="min-w-0 flex-1 pb-3">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">v{v.version}</span>
                            <span className="text-muted-foreground text-xs">
                              <IconCalendar className="mr-1 inline size-3" />
                              {formatDate(v.date)}
                            </span>
                          </div>
                          <p className="text-muted-foreground mt-0.5 text-sm">
                            {v.changes}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <Separator />

              <div className="flex items-center gap-2">
                {skill.installed ? (
                  <>
                    <div className="text-muted-foreground flex items-center gap-1 text-sm">
                      <IconCheck className="size-4 text-green-500" />
                      Installed
                    </div>
                    {skill.hasUpdate && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => onUpdate(skill.id)}
                        disabled={isUpdating}
                      >
                        {isUpdating ? (
                          <IconLoader2 className="size-4 animate-spin" />
                        ) : (
                          <IconUpload className="size-4" />
                        )}
                        Update
                      </Button>
                    )}
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => onUninstall(skill.id)}
                      disabled={isUninstalling}
                    >
                      {isUninstalling ? (
                        <IconLoader2 className="size-4 animate-spin" />
                      ) : (
                        <IconTrash className="size-4" />
                      )}
                      Uninstall
                    </Button>
                  </>
                ) : (
                  <Button
                    onClick={() => onInstall(skill.id)}
                    disabled={isInstalling}
                  >
                    {isInstalling ? (
                      <IconLoader2 className="size-4 animate-spin" />
                    ) : (
                      <IconDownload className="size-4" />
                    )}
                    Install
                  </Button>
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-6 p-6">
              <div className="space-y-3">
                <h3 className="text-sm font-medium">Leave a Review</h3>
                <div className="flex items-center gap-1">
                  {renderStars(userRating, true, setUserRating)}
                </div>
                <Textarea
                  placeholder="Share your experience with this skill..."
                  value={reviewComment}
                  onChange={(e) => setReviewComment(e.target.value)}
                  rows={3}
                />
                <Button
                  size="sm"
                  onClick={handleSubmitRating}
                  disabled={userRating === 0 || isSubmittingRating}
                >
                  {isSubmittingRating ? (
                    <IconLoader2 className="size-4 animate-spin" />
                  ) : (
                    "Submit Review"
                  )}
                </Button>
              </div>

              <Separator />

              <div className="space-y-4">
                <h3 className="text-sm font-medium">
                  Reviews ({reviews.length})
                </h3>
                {isLoadingReviews ? (
                  <div className="text-muted-foreground text-sm">Loading reviews...</div>
                ) : reviews.length === 0 ? (
                  <div className="text-muted-foreground text-sm">
                    No reviews yet. Be the first to review!
                  </div>
                ) : (
                  <div className="space-y-4">
                    {reviews.map((review: SkillReview) => (
                      <div key={review.id} className="space-y-2">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <span className="font-medium text-sm">{review.author}</span>
                            <div className="flex items-center gap-0.5">
                              {renderStars(review.rating)}
                            </div>
                          </div>
                          <span className="text-muted-foreground text-xs">
                            {formatDate(review.date)}
                          </span>
                        </div>
                        <p className="text-muted-foreground text-sm">
                          {review.comment}
                        </p>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
