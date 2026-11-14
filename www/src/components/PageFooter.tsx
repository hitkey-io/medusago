import React from "react"
import Link from "next/link"
import { ChatBubbleLeftRight } from "@medusajs/icons"
import { DottedSeparator, EditButton, Pagination, GITHUB_ISSUES_LINK } from "docs-ui"
import editDates from "~/generated/edit-dates.json"

// Same structure as the upstream docs footer: a separated "report" row, the previous/next
// cards (only in the sections that have them) and the edit date with the edit link.
// There is no thumbs-up/down widget because there is no analytics backend behind it.
export const PageFooter = ({ file, showPagination }: { file: string; showPagination: boolean }) => (
  <>
    <div className="my-docs_2">
      <DottedSeparator wrapperClassName="!my-docs_1 !px-0" />
      <div className="flex flex-wrap items-center gap-docs_1 text-compact-small-plus text-medusa-fg-base">
        <span>Found a problem with this page?</span>
        <Link
          href={GITHUB_ISSUES_LINK}
          className="flex items-center gap-[6px] px-docs_0.5 py-docs_0.25 rounded-docs_sm hover:bg-medusa-bg-base-hover"
        >
          <ChatBubbleLeftRight className="text-medusa-fg-subtle" />
          <span>Report Issue</span>
        </Link>
      </div>
      <DottedSeparator wrapperClassName="!my-docs_1 !px-0" />
    </div>
    {showPagination && <Pagination />}
    <EditButton filePath={file} editDate={(editDates as Record<string, string>)[file]} />
  </>
)
