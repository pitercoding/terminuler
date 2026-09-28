interface StepCardProps {
  step: number;
  title: string;
  children: React.ReactNode;
}

export function StepCard({ step, title, children }: StepCardProps) {
  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm sm:p-8">
      <h2 className="flex items-center gap-3 text-lg font-semibold text-slate-900">
        <span
          aria-hidden="true"
          className="flex size-7 items-center justify-center rounded-full bg-teal-50 text-sm font-semibold text-teal-700"
        >
          {step}
        </span>
        {title}
      </h2>

      <div className="mt-6">{children}</div>
    </section>
  );
}
