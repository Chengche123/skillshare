import { useEffect, useState } from 'react';
import { Copy } from 'lucide-react';
import Button from './Button';
import DialogShell from './DialogShell';
import { Input } from './Input';
import { useT } from '../i18n';

interface BulkCopySkillNamesDialogProps {
  open: boolean;
  selectedCount: number;
  onCopy: (delimiter: string) => void | Promise<void>;
  onClose: () => void;
}

export default function BulkCopySkillNamesDialog({
  open,
  selectedCount,
  onCopy,
  onClose,
}: BulkCopySkillNamesDialogProps) {
  const t = useT();
  const [delimiter, setDelimiter] = useState(', ');

  useEffect(() => {
    if (!open) return;
    setDelimiter(', ');
  }, [open, selectedCount]);

  return (
    <DialogShell open={open} onClose={onClose} maxWidth="md">
      <div className="flex items-start gap-3 mb-4">
        <div className="p-2 bg-muted text-pencil rounded-[var(--radius-sm)]">
          <Copy size={18} strokeWidth={2.5} />
        </div>
        <div className="min-w-0">
          <h3 className="text-lg font-bold text-pencil">
            {t('resources.bulk.copyNames', undefined, 'Copy names')}
          </h3>
          <p className="text-sm text-pencil-light">
            {t(
              'resources.bulk.copyNamesDescription',
              { count: selectedCount },
              `Copy ${selectedCount} selected skill names with a custom delimiter`,
            )}
          </p>
        </div>
      </div>

      <Input
        label={t('resources.bulk.delimiterLabel', undefined, 'Delimiter')}
        value={delimiter}
        onChange={(e) => setDelimiter(e.target.value)}
        placeholder={t('resources.bulk.delimiterPlaceholder', undefined, 'For example: , ')}
        autoFocus
      />

      <div className="flex justify-end gap-3 mt-6">
        <Button variant="secondary" size="md" onClick={onClose}>
          {t('common.cancel')}
        </Button>
        <Button variant="primary" size="md" onClick={() => onCopy(delimiter)}>
          <Copy size={16} strokeWidth={2.5} />
          {t('common.copy', undefined, 'Copy')}
        </Button>
      </div>
    </DialogShell>
  );
}
