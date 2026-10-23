import { defineComponent as C, computed as $, openBlock as a, createElementBlock as l, normalizeClass as B, toDisplayString as _, createCommentVNode as p, Fragment as M, renderList as z, createTextVNode as F, createElementVNode as r, unref as H, useAttrs as q, ref as D, mergeProps as U, watch as L, onMounted as Y, onBeforeUnmount as j, normalizeStyle as P, withModifiers as R, withKeys as G, renderSlot as E, createStaticVNode as X } from "vue";
function N(e) {
  const n = [], t = e ?? "";
  for (let u = 0; u < t.length; u++) {
    const s = t[u];
    if (s === "&") {
      const h = t[u + 1];
      if (h === "&") {
        n.push({ ch: "&", mnemonic: !1 }), u++;
        continue;
      }
      if (h !== void 0) {
        n.push({ ch: h, mnemonic: !0 }), u++;
        continue;
      }
      n.push({ ch: "&", mnemonic: !1 });
      continue;
    }
    n.push({ ch: s, mnemonic: !1 });
  }
  return n;
}
function J(e) {
  return N(e).map((n) => n.ch).join("");
}
const Q = ["disabled", "title"], Z = {
  key: 0,
  class: "w-btn__icon",
  "aria-hidden": "true"
}, ee = {
  key: 1,
  class: "w-btn__label"
}, te = {
  key: 0,
  class: "w-mnemonic"
}, ae = /* @__PURE__ */ C({
  __name: "WButton",
  props: {
    label: { default: "" },
    variant: { default: "push" },
    size: { default: "md" },
    disabled: { type: Boolean, default: !1 },
    active: { type: Boolean, default: !1 },
    icon: { default: "" },
    iconOnly: { type: Boolean, default: !1 },
    title: { default: "" }
  },
  emits: ["click", "dblclick", "focus", "blur"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => N(t.label)), h = $(() => !t.iconOnly && t.label.length > 0);
    return (b, i) => (a(), l("button", {
      type: "button",
      class: B(["w-btn w-selectable", [
        `w-btn--${e.variant}`,
        `w-btn--${e.size}`,
        { "is-active": e.active, "is-disabled": e.disabled }
      ]]),
      disabled: e.disabled,
      title: e.title || void 0,
      onClick: i[0] || (i[0] = (o) => u("click", o)),
      onDblclick: i[1] || (i[1] = (o) => u("dblclick", o)),
      onFocus: i[2] || (i[2] = (o) => u("focus", o)),
      onBlur: i[3] || (i[3] = (o) => u("blur", o))
    }, [
      e.icon ? (a(), l("span", Z, _(e.icon), 1)) : p("", !0),
      h.value ? (a(), l("span", ee, [
        (a(!0), l(M, null, z(s.value, (o, d) => (a(), l(M, { key: d }, [
          o.mnemonic ? (a(), l("u", te, _(o.ch), 1)) : (a(), l(M, { key: 1 }, [
            F(_(o.ch), 1)
          ], 64))
        ], 64))), 128))
      ])) : p("", !0)
    ], 42, Q));
  }
}), W = (e, n) => {
  const t = e.__vccOpts || e;
  for (const [u, s] of n)
    t[u] = s;
  return t;
}, le = /* @__PURE__ */ W(ae, [["__scopeId", "data-v-a507d20a"]]), ne = ["name", "checked", "indeterminate", "disabled"], ie = {
  class: "w-checkbox__box",
  "aria-hidden": "true"
}, se = {
  key: 0,
  class: "w-checkbox__dash"
}, oe = {
  key: 1,
  class: "w-checkbox__check"
}, de = { class: "w-checkbox__label" }, ue = {
  key: 0,
  class: "w-mnemonic"
}, ce = /* @__PURE__ */ C({
  __name: "WCheckBox",
  props: {
    modelValue: { type: [Boolean, String, null], default: !1 },
    label: { default: "" },
    triState: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 },
    name: { default: "" }
  },
  emits: ["update:modelValue", "change"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => t.triState && t.modelValue === "indeterminate" ? "indeterminate" : t.modelValue === !0);
    function h() {
      return t.triState ? s.value === !1 ? !0 : s.value === !0 ? "indeterminate" : !1 : !s.value;
    }
    function b() {
      const i = h();
      u("update:modelValue", i), u("change", i);
    }
    return (i, o) => (a(), l("label", {
      class: B(["w-checkbox w-selectable", { "is-disabled": e.disabled }])
    }, [
      r("input", {
        class: "w-checkbox__input",
        type: "checkbox",
        name: e.name || void 0,
        checked: s.value === !0,
        indeterminate: s.value === "indeterminate",
        disabled: e.disabled,
        onChange: b
      }, null, 40, ne),
      r("span", ie, [
        s.value === "indeterminate" ? (a(), l("span", se)) : s.value === !0 ? (a(), l("span", oe)) : p("", !0)
      ]),
      r("span", de, [
        (a(!0), l(M, null, z(H(N)(e.label), (d, c) => (a(), l(M, { key: c }, [
          d.mnemonic ? (a(), l("u", ue, _(d.ch), 1)) : (a(), l(M, { key: 1 }, [
            F(_(d.ch), 1)
          ], 64))
        ], 64))), 128))
      ])
    ], 2));
  }
}), re = /* @__PURE__ */ W(ce, [["__scopeId", "data-v-a243fb34"]]), he = ["name", "value", "checked", "disabled"], fe = {
  class: "w-radio__dot",
  "aria-hidden": "true"
}, me = {
  key: 0,
  class: "w-radio__fill"
}, be = { class: "w-radio__label" }, ve = {
  key: 0,
  class: "w-mnemonic"
}, we = /* @__PURE__ */ C({
  __name: "WRadioButton",
  props: {
    modelValue: { default: null },
    value: {},
    label: { default: "" },
    name: { default: "" },
    disabled: { type: Boolean, default: !1 }
  },
  emits: ["update:modelValue", "change"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => t.modelValue === t.value);
    function h() {
      u("update:modelValue", t.value), u("change", t.value);
    }
    return (b, i) => (a(), l("label", {
      class: B(["w-radio w-selectable", { "is-disabled": e.disabled }])
    }, [
      r("input", {
        class: "w-radio__input",
        type: "radio",
        name: e.name || void 0,
        value: e.value,
        checked: s.value,
        disabled: e.disabled,
        onChange: h
      }, null, 40, he),
      r("span", fe, [
        s.value ? (a(), l("span", me)) : p("", !0)
      ]),
      r("span", be, [
        (a(!0), l(M, null, z(H(N)(e.label), (o, d) => (a(), l(M, { key: d }, [
          o.mnemonic ? (a(), l("u", ve, _(o.ch), 1)) : (a(), l(M, { key: 1 }, [
            F(_(o.ch), 1)
          ], 64))
        ], 64))), 128))
      ])
    ], 2));
  }
}), ye = /* @__PURE__ */ W(we, [["__scopeId", "data-v-f501c7e7"]]), ge = ["type", "value", "placeholder", "maxlength", "readonly", "disabled"], _e = /* @__PURE__ */ C({
  __name: "WEdit",
  props: {
    modelValue: { default: "" },
    type: { default: "text" },
    readonly: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 },
    placeholder: { default: "" },
    maxlength: { default: void 0 },
    align: { default: "left" },
    error: { type: Boolean, default: !1 },
    invalid: { type: Boolean, default: !1 },
    width: { default: "100%" }
  },
  emits: ["update:modelValue", "enter", "blur", "focus"],
  setup(e, { expose: n, emit: t }) {
    const u = e, s = t, h = q(), b = D(null), i = $(() => u.error || u.invalid);
    function o(y) {
      const x = y.target;
      s("update:modelValue", x.value);
    }
    function d(y) {
      y.key === "Enter" && s("enter");
    }
    function c() {
      var y;
      (y = b.value) == null || y.focus();
    }
    return n({ focus: c, el: b }), (y, x) => (a(), l("input", U({
      ref_key: "inner",
      ref: b
    }, H(h), {
      class: ["w-edit", { "is-disabled": e.disabled, "is-readonly": e.readonly, "is-error": i.value }],
      type: e.type === "number" ? "text" : e.type,
      value: e.modelValue,
      placeholder: e.placeholder || void 0,
      maxlength: e.maxlength,
      readonly: e.readonly,
      disabled: e.disabled,
      style: { width: typeof e.width == "number" ? e.width + "px" : e.width, textAlign: e.align },
      onInput: o,
      onKeydown: d,
      onBlur: x[0] || (x[0] = (w) => s("blur", w)),
      onFocus: x[1] || (x[1] = (w) => s("focus", w))
    }), null, 16, ge));
  }
}), pe = /* @__PURE__ */ W(_e, [["__scopeId", "data-v-b9f59322"]]), xe = ["value", "rows", "cols", "readonly", "disabled", "placeholder", "wrap"], ke = /* @__PURE__ */ C({
  __name: "WTextArea",
  props: {
    modelValue: { default: "" },
    rows: { default: 4 },
    cols: { default: 20 },
    readonly: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 },
    wrap: { default: "soft" },
    placeholder: { default: "" }
  },
  emits: ["update:modelValue"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = q(), h = $(() => t.wrap === "off" ? "off" : t.wrap);
    function b(i) {
      u("update:modelValue", i.target.value);
    }
    return (i, o) => (a(), l("textarea", U(H(s), {
      class: ["w-textarea", { "is-disabled": e.disabled, "is-readonly": e.readonly }],
      value: e.modelValue,
      rows: e.rows,
      cols: e.cols,
      readonly: e.readonly,
      disabled: e.disabled,
      placeholder: e.placeholder || void 0,
      wrap: h.value,
      onInput: b
    }), null, 16, xe));
  }
}), $e = /* @__PURE__ */ W(ke, [["__scopeId", "data-v-5f5569aa"]]), Me = { class: "w-combo__control" }, Be = ["value", "disabled", "placeholder"], Ve = ["disabled"], We = {
  key: 0,
  class: "w-combo__list w-bevel-raised",
  role: "listbox"
}, Ce = ["aria-selected", "onClick"], Se = {
  key: 0,
  class: "w-combo__item is-empty"
}, Ie = /* @__PURE__ */ C({
  __name: "WComboBox",
  props: {
    modelValue: { default: null },
    options: { default: () => [] },
    editable: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 },
    width: { default: "160px" },
    placeholder: { default: "" }
  },
  emits: ["update:modelValue", "change", "open", "close"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(!1), h = D(null), b = D(""), i = $(
      () => t.options.find((w) => w.value === t.modelValue) ?? null
    ), o = $(() => t.editable ? b.value : i.value ? i.value.label : "");
    L(
      () => t.modelValue,
      () => {
        !t.editable && i.value && (b.value = i.value.label);
      },
      { immediate: !0 }
    ), L(
      () => t.editable,
      (w) => {
        w && i.value && (b.value = i.value.label);
      }
    );
    function d() {
      t.disabled || (s.value = !s.value, s.value ? u("open") : u("close"));
    }
    function c(w) {
      w.disabled || (t.editable || (u("update:modelValue", w.value), u("change", w.value)), s.value = !1, u("close"));
    }
    function y(w) {
      b.value = w.target.value;
    }
    function x(w) {
      s.value && h.value && !h.value.contains(w.target) && (s.value = !1, u("close"));
    }
    return Y(() => document.addEventListener("mousedown", x)), j(() => document.removeEventListener("mousedown", x)), (w, v) => (a(), l("div", {
      ref_key: "root",
      ref: h,
      class: "w-combo",
      style: P({ width: typeof e.width == "number" ? e.width + "px" : e.width })
    }, [
      r("div", Me, [
        e.editable ? (a(), l("input", {
          key: 0,
          class: "w-combo__input",
          value: b.value,
          disabled: e.disabled,
          placeholder: e.placeholder || void 0,
          onInput: y
        }, null, 40, Be)) : (a(), l("span", {
          key: 1,
          class: B(["w-combo__text", { "is-empty": !o.value }])
        }, _(o.value || e.placeholder), 3)),
        r("button", {
          type: "button",
          class: B(["w-combo__arrow", { "is-open": s.value, "is-disabled": e.disabled }]),
          disabled: e.disabled,
          tabindex: "-1",
          "aria-label": "Open dropdown",
          onMousedown: v[0] || (v[0] = R(() => {
          }, ["prevent"])),
          onClick: d
        }, [...v[1] || (v[1] = [
          r("span", { class: "w-combo__arrow-glyph" }, null, -1)
        ])], 42, Ve)
      ]),
      s.value ? (a(), l("ul", We, [
        (a(!0), l(M, null, z(e.options, (g, f) => (a(), l("li", {
          key: f,
          class: B(["w-combo__item", {
            "is-selected": g.value === e.modelValue,
            "is-disabled": g.disabled
          }]),
          role: "option",
          "aria-selected": g.value === e.modelValue,
          onClick: (m) => c(g)
        }, _(g.label), 11, Ce))), 128)),
        e.options.length === 0 ? (a(), l("li", Se, "(no items)")) : p("", !0)
      ])) : p("", !0)
    ], 4));
  }
}), Pe = /* @__PURE__ */ W(Ie, [["__scopeId", "data-v-171f5d58"]]), Te = ["aria-multiselectable"], ze = ["aria-selected", "onClick"], De = {
  key: 0,
  class: "w-listbox__item is-empty"
}, Ae = /* @__PURE__ */ C({
  __name: "WListBox",
  props: {
    modelValue: { default: null },
    options: { default: () => [] },
    multiple: { type: Boolean, default: !1 },
    height: { default: "120px" },
    disabled: { type: Boolean, default: !1 },
    width: { default: "100%" }
  },
  emits: ["update:modelValue", "change"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => {
      if (t.multiple) return Array.isArray(t.modelValue) ? t.modelValue : [];
      const i = t.modelValue;
      return i == null || Array.isArray(i) ? [] : [i];
    });
    function h(i) {
      return s.value.includes(i);
    }
    function b(i) {
      if (t.disabled || i.disabled) return;
      let o;
      t.multiple ? o = h(i.value) ? s.value.filter((d) => d !== i.value) : [...s.value, i.value] : o = [i.value], u("update:modelValue", t.multiple ? o : o[0]), u("change", t.multiple ? o : o[0]);
    }
    return (i, o) => (a(), l("ul", {
      class: B(["w-listbox", { "is-disabled": e.disabled }]),
      style: P({
        height: typeof e.height == "number" ? e.height + "px" : e.height,
        width: typeof e.width == "number" ? e.width + "px" : e.width
      }),
      role: "listbox",
      "aria-multiselectable": e.multiple || void 0
    }, [
      (a(!0), l(M, null, z(e.options, (d, c) => (a(), l("li", {
        key: c,
        class: B(["w-listbox__item", {
          "is-selected": h(d.value),
          "is-disabled": d.disabled
        }]),
        role: "option",
        "aria-selected": h(d.value),
        onClick: (y) => b(d)
      }, _(d.label), 11, ze))), 128)),
      e.options.length === 0 ? (a(), l("li", De, "(empty)")) : p("", !0)
    ], 14, Te));
  }
}), Ee = /* @__PURE__ */ W(Ae, [["__scopeId", "data-v-d9050eaa"]]), Le = { class: "w-updown__field" }, Fe = { class: "w-updown__text w-mono" }, Ne = { class: "w-updown__buttons" }, Re = ["disabled"], Oe = ["disabled"], Ke = /* @__PURE__ */ C({
  __name: "WUpDown",
  props: {
    modelValue: { default: 0 },
    min: { default: 0 },
    max: { default: 100 },
    step: { default: 1 },
    disabled: { type: Boolean, default: !1 },
    width: { default: "52px" }
  },
  emits: ["update:modelValue", "change"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => t.modelValue);
    function h(i) {
      return Math.min(t.max, Math.max(t.min, i));
    }
    function b(i) {
      if (t.disabled) return;
      const o = h(Number((s.value + i).toFixed(6)));
      o !== s.value && (u("update:modelValue", o), u("change", o));
    }
    return (i, o) => (a(), l("div", {
      class: B(["w-updown", { "is-disabled": e.disabled }]),
      style: P({ width: typeof e.width == "number" ? e.width + "px" : e.width })
    }, [
      r("div", Le, [
        r("span", Fe, _(s.value), 1)
      ]),
      r("div", Ne, [
        r("button", {
          type: "button",
          class: "w-updown__btn w-updown__btn--up",
          disabled: e.disabled,
          tabindex: "-1",
          "aria-label": "Increase",
          onClick: o[0] || (o[0] = (d) => b(e.step))
        }, [...o[2] || (o[2] = [
          r("span", { class: "w-updown__glyph w-updown__glyph--up" }, null, -1)
        ])], 8, Re),
        r("button", {
          type: "button",
          class: "w-updown__btn w-updown__btn--down",
          disabled: e.disabled,
          tabindex: "-1",
          "aria-label": "Decrease",
          onClick: o[1] || (o[1] = (d) => b(-e.step))
        }, [...o[3] || (o[3] = [
          r("span", { class: "w-updown__glyph w-updown__glyph--down" }, null, -1)
        ])], 8, Oe)
      ])
    ], 6));
  }
}), Ye = /* @__PURE__ */ W(Ke, [["__scopeId", "data-v-892ffb0c"]]), He = ["aria-valuemin", "aria-valuemax", "aria-valuenow", "aria-label"], je = {
  key: 0,
  class: "w-trackbar__ticks"
}, Ue = /* @__PURE__ */ C({
  __name: "WTrackbar",
  props: {
    modelValue: { default: 0 },
    min: { default: 0 },
    max: { default: 100 },
    step: { default: 1 },
    orientation: { default: "horizontal" },
    ticks: { type: Boolean, default: !0 },
    disabled: { type: Boolean, default: !1 },
    label: { default: "" }
  },
  emits: ["update:modelValue", "change"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(null), h = D(!1), b = $(() => {
      const f = t.max - t.min;
      return f <= 0 ? 0 : Math.min(1, Math.max(0, (t.modelValue - t.min) / f));
    }), i = $(() => `${b.value * 100}%`), o = $(() => {
      const f = t.max - t.min;
      if (f <= 0) return 2;
      const m = Math.round(f / 10);
      return Math.min(11, Math.max(2, m + 1));
    });
    function d(f) {
      const m = s.value;
      if (!m) return t.modelValue;
      const V = m.getBoundingClientRect(), T = V.width > 0 ? (f - V.left) / V.width : 0, k = t.min + Math.min(1, Math.max(0, T)) * (t.max - t.min);
      return y(k);
    }
    function c(f) {
      const m = s.value;
      if (!m) return t.modelValue;
      const V = m.getBoundingClientRect(), T = V.height > 0 ? (V.bottom - f) / V.height : 0, k = t.min + Math.min(1, Math.max(0, T)) * (t.max - t.min);
      return y(k);
    }
    function y(f) {
      const m = t.step > 0 ? t.step : 1, V = Math.round((f - t.min) / m) * m + t.min;
      return Math.min(t.max, Math.max(t.min, Number(V.toFixed(6))));
    }
    function x(f) {
      var V, T;
      if (t.disabled) return;
      h.value = !0, (T = (V = f.currentTarget).setPointerCapture) == null || T.call(V, f.pointerId);
      const m = t.orientation === "horizontal" ? d(f.clientX) : c(f.clientY);
      u("update:modelValue", m);
    }
    function w(f) {
      if (!h.value || t.disabled) return;
      const m = t.orientation === "horizontal" ? d(f.clientX) : c(f.clientY);
      m !== t.modelValue && u("update:modelValue", m);
    }
    function v(f) {
      var m, V;
      h.value && (h.value = !1, (V = (m = f.currentTarget).releasePointerCapture) == null || V.call(m, f.pointerId), u("change", t.modelValue));
    }
    function g(f) {
      if (t.disabled) return;
      let m = null;
      const V = (t.max - t.min) / 10;
      (f.key === "ArrowLeft" || f.key === "ArrowDown") && (m = y(t.modelValue - t.step)), (f.key === "ArrowRight" || f.key === "ArrowUp") && (m = y(t.modelValue + t.step)), f.key === "PageDown" && (m = y(t.modelValue - V)), f.key === "PageUp" && (m = y(t.modelValue + V)), f.key === "Home" && (m = t.min), f.key === "End" && (m = t.max), m !== null && (f.preventDefault(), u("update:modelValue", m), u("change", m));
    }
    return (f, m) => (a(), l("div", {
      class: B(["w-trackbar", [`w-trackbar--${e.orientation}`, { "is-disabled": e.disabled }]])
    }, [
      r("div", {
        ref_key: "track",
        ref: s,
        class: "w-trackbar__track",
        role: "slider",
        tabindex: "0",
        "aria-valuemin": e.min,
        "aria-valuemax": e.max,
        "aria-valuenow": e.modelValue,
        "aria-label": e.label || void 0,
        onPointerdown: x,
        onPointermove: w,
        onPointerup: v,
        onPointercancel: v,
        onKeydown: g
      }, [
        r("div", {
          class: "w-trackbar__thumb",
          style: P({ [e.orientation === "horizontal" ? "left" : "bottom"]: i.value })
        }, null, 4)
      ], 40, He),
      e.ticks ? (a(), l("div", je, [
        (a(!0), l(M, null, z(o.value, (V) => (a(), l("span", {
          key: V,
          class: "w-trackbar__tick",
          style: P({ [e.orientation === "horizontal" ? "left" : "bottom"]: `${(V - 1) / Math.max(1, o.value - 1) * 100}%` })
        }, null, 4))), 128))
      ])) : p("", !0)
    ], 2));
  }
}), qe = /* @__PURE__ */ W(Ue, [["__scopeId", "data-v-fce06e73"]]), Ge = { class: "w-datetime__field" }, Xe = ["value", "disabled", "placeholder"], Je = ["disabled"], Qe = {
  key: 0,
  class: "w-datetime__popup w-bevel-raised"
}, Ze = { class: "w-datetime__head" }, et = { class: "w-datetime__title" }, tt = { class: "w-datetime__grid" }, at = ["onClick"], lt = /* @__PURE__ */ C({
  __name: "WDateTime",
  props: {
    modelValue: { default: "" },
    format: { default: "yyyy-MM-dd" },
    showTime: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 },
    width: { default: "140px" }
  },
  emits: ["update:modelValue", "change"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(!1), h = D(null);
    function b(k) {
      if (!k) return null;
      const S = k.match(/^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/);
      if (!S) return null;
      const I = new Date(
        Number(S[1]),
        Number(S[2]) - 1,
        Number(S[3]),
        Number(S[4] ?? 0),
        Number(S[5] ?? 0),
        Number(S[6] ?? 0)
      );
      return Number.isNaN(I.getTime()) ? null : I;
    }
    const i = $(() => b(t.modelValue) ?? /* @__PURE__ */ new Date()), o = D(i.value.getFullYear()), d = D(i.value.getMonth()), c = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"], y = [
      "January",
      "February",
      "March",
      "April",
      "May",
      "June",
      "July",
      "August",
      "September",
      "October",
      "November",
      "December"
    ], x = $(() => {
      const S = (new Date(o.value, d.value, 1).getDay() + 6) % 7, I = [], A = new Date(o.value, d.value, 1 - S);
      for (let O = 0; O < 42; O++) {
        const K = new Date(A.getFullYear(), A.getMonth(), A.getDate() + O);
        I.push({
          day: K.getDate(),
          inMonth: K.getMonth() === d.value,
          iso: v(K)
        });
      }
      return I;
    });
    function w(k) {
      return String(k).padStart(2, "0");
    }
    function v(k) {
      return `${k.getFullYear()}-${w(k.getMonth() + 1)}-${w(k.getDate())}`;
    }
    function g(k) {
      const S = `${k.getFullYear()}-${w(k.getMonth() + 1)}-${w(k.getDate())}`;
      return t.showTime ? `${S} ${w(k.getHours())}:${w(k.getMinutes())}:${w(k.getSeconds())}` : S;
    }
    function f(k) {
      const S = b(k.iso) ?? /* @__PURE__ */ new Date();
      if (t.showTime) {
        const A = i.value;
        S.setHours(A.getHours(), A.getMinutes(), A.getSeconds());
      }
      const I = g(S);
      u("update:modelValue", I), u("change", I), s.value = !1;
    }
    function m() {
      const k = /* @__PURE__ */ new Date();
      o.value = k.getFullYear(), d.value = k.getMonth(), f({ day: k.getDate(), iso: v(k) });
    }
    function V(k) {
      const S = new Date(o.value, d.value + k, 1);
      o.value = S.getFullYear(), d.value = S.getMonth();
    }
    function T(k) {
      s.value && h.value && !h.value.contains(k.target) && (s.value = !1);
    }
    return Y(() => document.addEventListener("mousedown", T)), j(() => document.removeEventListener("mousedown", T)), (k, S) => (a(), l("div", {
      ref_key: "root",
      ref: h,
      class: "w-datetime",
      style: P({ width: typeof e.width == "number" ? e.width + "px" : e.width })
    }, [
      r("div", Ge, [
        r("input", {
          class: "w-datetime__input w-mono",
          value: e.modelValue,
          disabled: e.disabled,
          placeholder: e.format,
          onInput: S[0] || (S[0] = (I) => u("update:modelValue", I.target.value))
        }, null, 40, Xe)
      ]),
      r("button", {
        type: "button",
        class: "w-datetime__btn",
        disabled: e.disabled,
        tabindex: "-1",
        "aria-label": "Open calendar",
        onMousedown: S[1] || (S[1] = R(() => {
        }, ["prevent"])),
        onClick: S[2] || (S[2] = (I) => s.value = !s.value)
      }, [...S[5] || (S[5] = [
        r("span", { class: "w-datetime__glyph" }, null, -1)
      ])], 40, Je),
      s.value ? (a(), l("div", Qe, [
        r("div", Ze, [
          r("button", {
            type: "button",
            class: "w-datetime__nav",
            onClick: S[3] || (S[3] = (I) => V(-1))
          }, "◀"),
          r("span", et, _(y[d.value]) + " " + _(o.value), 1),
          r("button", {
            type: "button",
            class: "w-datetime__nav",
            onClick: S[4] || (S[4] = (I) => V(1))
          }, "▶")
        ]),
        r("table", tt, [
          r("thead", null, [
            r("tr", null, [
              (a(), l(M, null, z(c, (I) => r("th", { key: I }, _(I), 1)), 64))
            ])
          ]),
          r("tbody", null, [
            r("tr", null, [
              (a(!0), l(M, null, z(x.value, (I) => (a(), l("td", {
                key: I.iso,
                class: B({ "is-out": !I.inMonth, "is-today": I.iso === v(/* @__PURE__ */ new Date()) }),
                onClick: (A) => f(I)
              }, _(I.day), 11, at))), 128))
            ])
          ])
        ]),
        r("div", { class: "w-datetime__foot" }, [
          r("button", {
            type: "button",
            class: "w-datetime__today",
            onClick: m
          }, "Today")
        ])
      ])) : p("", !0)
    ], 4));
  }
}), nt = /* @__PURE__ */ W(lt, [["__scopeId", "data-v-f1fd6d75"]]), it = ["value", "placeholder", "disabled"], st = ["disabled"], ot = /* @__PURE__ */ C({
  __name: "WSearchBox",
  props: {
    modelValue: { default: "" },
    placeholder: { default: "Search" },
    disabled: { type: Boolean, default: !1 },
    width: { default: "180px" }
  },
  emits: ["update:modelValue", "search", "clear"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => t.modelValue.length > 0);
    function h() {
      u("update:modelValue", ""), u("clear"), u("search", "");
    }
    return (b, i) => (a(), l("div", {
      class: B(["w-searchbox", { "is-disabled": e.disabled }]),
      style: P({ width: typeof e.width == "number" ? e.width + "px" : e.width })
    }, [
      i[2] || (i[2] = r("span", {
        class: "w-searchbox__glass",
        "aria-hidden": "true"
      }, null, -1)),
      r("input", {
        class: "w-searchbox__input",
        type: "text",
        value: e.modelValue,
        placeholder: e.placeholder,
        disabled: e.disabled,
        onInput: i[0] || (i[0] = (o) => u("update:modelValue", o.target.value)),
        onKeydown: i[1] || (i[1] = G((o) => u("search", e.modelValue), ["enter"]))
      }, null, 40, it),
      s.value ? (a(), l("button", {
        key: 0,
        type: "button",
        class: "w-searchbox__clear",
        disabled: e.disabled,
        tabindex: "-1",
        "aria-label": "Clear",
        onClick: h
      }, " × ", 8, st)) : p("", !0)
    ], 6));
  }
}), dt = /* @__PURE__ */ W(ot, [["__scopeId", "data-v-c746c063"]]), ut = {
  key: 0,
  class: "w-listview__head"
}, ct = ["onClick"], rt = { class: "w-listview__col-title" }, ht = {
  key: 0,
  class: "w-listview__sortmark"
}, ft = ["onClick"], mt = {
  key: 0,
  class: "w-listview__empty"
}, bt = /* @__PURE__ */ C({
  __name: "WListView",
  props: {
    columns: { default: () => [] },
    rows: { default: () => [] },
    modelValue: { default: null },
    selectable: { type: Boolean, default: !0 },
    multiSelect: { type: Boolean, default: !1 },
    gridLines: { type: Boolean, default: !1 },
    rowHeight: { default: 18 },
    showHeader: { type: Boolean, default: !0 },
    height: { default: "100%" }
  },
  emits: ["update:modelValue", "rowClick", "sort"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(""), h = D("asc"), b = $(() => Array.isArray(t.modelValue) ? t.modelValue : t.modelValue === null || t.modelValue === void 0 ? [] : [t.modelValue]), i = $(() => {
      if (!s.value) return t.rows;
      t.columns.find((v) => v.key === s.value);
      const w = h.value === "asc" ? 1 : -1;
      return [...t.rows].sort((v, g) => {
        const f = v[s.value], m = g[s.value];
        return typeof f == "number" && typeof m == "number" ? (f - m) * w : String(f ?? "").localeCompare(String(m ?? "")) * w;
      });
    });
    function o(w, v) {
      const g = t.columns[0];
      return g && w[g.key] !== void 0 ? String(w[g.key]) : `row-${v}`;
    }
    function d(w, v) {
      return b.value.includes(o(w, v));
    }
    function c(w, v, g) {
      if (!t.selectable) {
        u("rowClick", w, v);
        return;
      }
      const f = o(w, v);
      let m;
      if (t.multiSelect) {
        const V = g.ctrlKey || g.metaKey ? [...b.value] : [], T = V.indexOf(f);
        T >= 0 ? V.splice(T, 1) : V.push(f), m = V;
      } else
        m = f;
      u("update:modelValue", m), u("rowClick", w, v);
    }
    function y(w) {
      w.sortable && (s.value === w.key ? h.value = h.value === "asc" ? "desc" : "asc" : (s.value = w.key, h.value = "asc"), u("sort", w.key, h.value));
    }
    function x(w) {
      return w.width === void 0 ? "auto" : typeof w.width == "number" ? `${w.width}px` : w.width;
    }
    return (w, v) => (a(), l("div", {
      class: "w-listview",
      style: P({ height: typeof e.height == "number" ? e.height + "px" : e.height })
    }, [
      e.showHeader ? (a(), l("div", ut, [
        (a(!0), l(M, null, z(e.columns, (g) => (a(), l("div", {
          key: g.key,
          class: B(["w-listview__col", [`is-${g.align ?? "left"}`, { "is-sortable": g.sortable }]]),
          style: P({ width: x(g) }),
          onClick: (f) => y(g)
        }, [
          r("span", rt, _(g.title), 1),
          s.value === g.key ? (a(), l("span", ht, _(h.value === "asc" ? "▲" : "▼"), 1)) : p("", !0)
        ], 14, ct))), 128))
      ])) : p("", !0),
      r("div", {
        class: B(["w-listview__body", { "has-grid": e.gridLines }])
      }, [
        (a(!0), l(M, null, z(i.value, (g, f) => (a(), l("div", {
          key: o(g, f),
          class: B(["w-listview__row", { "is-selected": d(g, f) }]),
          style: P({ height: `${e.rowHeight}px`, lineHeight: `${e.rowHeight}px` }),
          onClick: (m) => c(g, f, m)
        }, [
          (a(!0), l(M, null, z(e.columns, (m) => (a(), l("div", {
            key: m.key,
            class: B(["w-listview__cell", `is-${m.align ?? "left"}`])
          }, _(g[m.key] ?? ""), 3))), 128))
        ], 14, ft))), 128)),
        i.value.length === 0 ? (a(), l("div", mt, "(no records)")) : p("", !0)
      ], 2)
    ], 4));
  }
}), vt = /* @__PURE__ */ W(bt, [["__scopeId", "data-v-db66f89d"]]), wt = ["aria-expanded", "onClick"], yt = {
  key: 0,
  class: "w-treeview__lines",
  "aria-hidden": "true"
}, gt = ["onClick"], _t = {
  key: 0,
  class: "w-treeview__toggle-sign"
}, pt = {
  key: 1,
  class: "w-treeview__icon"
}, xt = { class: "w-treeview__label" }, kt = {
  key: 0,
  class: "w-treeview__empty"
}, $t = /* @__PURE__ */ C({
  __name: "WTreeView",
  props: {
    nodes: { default: () => [] },
    modelValue: { default: null },
    showLines: { type: Boolean, default: !0 },
    expandOnClick: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 },
    height: { default: "100%" }
  },
  emits: ["update:modelValue", "nodeClick", "toggle"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(/* @__PURE__ */ new Set());
    function h(d, c) {
      for (const y of d)
        y.children && y.children.length && (c.add(y.id), h(y.children, c));
    }
    L(
      () => t.nodes,
      (d) => {
        const c = /* @__PURE__ */ new Set();
        h(d, c), s.value = c;
      },
      { immediate: !0, deep: !0 }
    );
    const b = $(() => {
      const d = [], c = (y, x) => {
        y.forEach((w, v) => {
          const g = !!(w.children && w.children.length), f = s.value.has(w.id);
          d.push({ node: w, depth: x, hasChildren: g, isExpanded: f, isLast: v === y.length - 1 }), g && f && c(w.children, x + 1);
        });
      };
      return c(t.nodes, 0), d;
    });
    function i(d) {
      if (!d.children || !d.children.length) return;
      const c = new Set(s.value), y = c.has(d.id);
      y ? c.delete(d.id) : c.add(d.id), s.value = c, u("toggle", d, !y);
    }
    function o(d) {
      u("update:modelValue", d.id), u("nodeClick", d), t.expandOnClick && i(d);
    }
    return (d, c) => (a(), l("div", {
      class: "w-treeview",
      style: P({ height: typeof e.height == "number" ? e.height + "px" : e.height }),
      role: "tree"
    }, [
      (a(!0), l(M, null, z(b.value, (y) => (a(), l("div", {
        key: y.node.id,
        class: B(["w-treeview__row", { "is-selected": e.modelValue === y.node.id }]),
        style: P({ paddingLeft: `${y.depth * 16 + 3}px` }),
        role: "treeitem",
        "aria-expanded": y.hasChildren ? y.isExpanded : void 0,
        onClick: (x) => o(y.node)
      }, [
        e.showLines ? (a(), l("span", yt, [
          (a(!0), l(M, null, z(y.depth, (x) => (a(), l("span", {
            key: x,
            class: "w-treeview__line"
          }))), 128))
        ])) : p("", !0),
        r("span", {
          class: B(["w-treeview__toggle", { "is-empty": !y.hasChildren, "is-open": y.isExpanded }]),
          onClick: R((x) => i(y.node), ["stop"])
        }, [
          y.hasChildren ? (a(), l("span", _t)) : p("", !0)
        ], 10, gt),
        y.node.icon ? (a(), l("span", pt, _(y.node.icon), 1)) : p("", !0),
        r("span", xt, _(y.node.label), 1)
      ], 14, wt))), 128)),
      b.value.length === 0 ? (a(), l("div", kt, "(empty)")) : p("", !0)
    ], 4));
  }
}), Mt = /* @__PURE__ */ W($t, [["__scopeId", "data-v-7e067060"]]), Bt = ["aria-valuemax", "aria-valuenow"], Vt = /* @__PURE__ */ C({
  __name: "WProgressBar",
  props: {
    value: { default: 0 },
    max: { default: 100 },
    indeterminate: { type: Boolean, default: !1 },
    segments: { default: 0 },
    width: { default: "100%" },
    height: { default: 14 }
  },
  setup(e) {
    const n = e, t = D(null), u = D(0);
    let s;
    const h = $(() => n.indeterminate ? 1 : n.max <= 0 ? 0 : Math.min(1, Math.max(0, n.value / n.max))), b = $(() => n.segments > 0 ? n.segments : 10), i = $(() => n.indeterminate ? 1 : n.segments > 0 ? Math.round(h.value * n.segments) / n.segments : h.value), o = $(() => {
      const d = b.value;
      return u.value ? Math.max(4, Math.floor(u.value / d)) : 10;
    });
    return Y(() => {
      typeof ResizeObserver > "u" || (s = new ResizeObserver(() => {
        var d;
        u.value = ((d = t.value) == null ? void 0 : d.clientWidth) ?? 0;
      }), t.value && s.observe(t.value));
    }), j(() => s == null ? void 0 : s.disconnect()), (d, c) => (a(), l("div", {
      ref_key: "track",
      ref: t,
      class: B(["w-progress", { "is-indeterminate": e.indeterminate }]),
      style: P({
        width: typeof e.width == "number" ? e.width + "px" : e.width,
        height: `${e.height}px`
      }),
      role: "progressbar",
      "aria-valuemin": 0,
      "aria-valuemax": e.max,
      "aria-valuenow": e.indeterminate ? void 0 : e.value
    }, [
      r("div", {
        class: "w-progress__fill",
        style: P({
          width: `${i.value * 100}%`,
          backgroundImage: `repeating-linear-gradient(90deg, var(--w-progress) 0 ${o.value - 1}px, transparent ${o.value - 1}px ${o.value}px)`
        })
      }, null, 4)
    ], 14, Bt));
  }
}), Wt = /* @__PURE__ */ W(Vt, [["__scopeId", "data-v-d660d63f"]]), Ct = ["for", "title"], St = {
  key: 0,
  class: "w-mnemonic"
}, It = /* @__PURE__ */ C({
  __name: "WLabel",
  props: {
    text: { default: "" },
    align: { default: "left" },
    disabled: { type: Boolean, default: !1 },
    for: { default: "" },
    bold: { type: Boolean, default: !1 }
  },
  setup(e) {
    const n = e, t = $(() => N(n.text)), u = $(() => J(n.text)), s = $(() => n.for || void 0);
    return (h, b) => (a(), l("label", {
      class: B(["w-label", [`is-${e.align}`, { "is-disabled": e.disabled, "is-bold": e.bold }]]),
      for: s.value,
      title: u.value
    }, [
      E(h.$slots, "default", {}, () => [
        (a(!0), l(M, null, z(t.value, (i, o) => (a(), l(M, { key: o }, [
          i.mnemonic ? (a(), l("u", St, _(i.ch), 1)) : (a(), l(M, { key: 1 }, [
            F(_(i.ch), 1)
          ], 64))
        ], 64))), 128))
      ], !0)
    ], 10, Ct));
  }
}), Pt = /* @__PURE__ */ W(It, [["__scopeId", "data-v-9084a4a4"]]), Tt = ["href", "aria-disabled"], zt = /* @__PURE__ */ C({
  __name: "WLink",
  props: {
    href: { default: "#" },
    disabled: { type: Boolean, default: !1 }
  },
  emits: ["click"],
  setup(e, { emit: n }) {
    const t = n;
    return (u, s) => (a(), l("a", {
      class: B(["w-link", { "is-disabled": e.disabled }]),
      href: e.disabled ? void 0 : e.href,
      "aria-disabled": e.disabled || void 0,
      onClick: s[0] || (s[0] = (h) => !e.disabled && t("click", h))
    }, [
      E(u.$slots, "default", {}, void 0, !0)
    ], 10, Tt));
  }
}), Dt = /* @__PURE__ */ W(zt, [["__scopeId", "data-v-b267c225"]]), At = ["aria-orientation"], Et = {
  key: 0,
  class: "w-separator__label"
}, Lt = /* @__PURE__ */ C({
  __name: "WSeparator",
  props: {
    orientation: { default: "horizontal" },
    inset: { type: Boolean, default: !1 },
    label: { default: "" }
  },
  setup(e) {
    return (n, t) => (a(), l("div", {
      class: B(["w-separator", [`w-separator--${e.orientation}`, { "is-inset": e.inset }]]),
      role: "separator",
      "aria-orientation": e.orientation
    }, [
      e.label ? (a(), l("span", Et, _(e.label), 1)) : p("", !0)
    ], 10, At));
  }
}), Ft = /* @__PURE__ */ W(Lt, [["__scopeId", "data-v-333c77d8"]]), Nt = ["title"], Rt = ["src"], Ot = /* @__PURE__ */ C({
  __name: "WIcon",
  props: {
    src: { default: "" },
    glyph: { default: "" },
    size: { default: 16 },
    shape: { default: "none" },
    title: { default: "" },
    disabled: { type: Boolean, default: !1 }
  },
  setup(e) {
    const n = e, t = $(() => (typeof n.size == "number", n.size));
    return (u, s) => (a(), l("span", {
      class: B(["w-icon", [`w-icon--${e.shape}`, { "is-disabled": e.disabled }]]),
      style: P({ width: `${t.value}px`, height: `${t.value}px` }),
      title: e.title || void 0,
      "aria-hidden": "true"
    }, [
      e.src ? (a(), l("img", {
        key: 0,
        class: "w-icon__img",
        src: e.src,
        alt: "",
        style: P({ width: `${t.value}px`, height: `${t.value}px` })
      }, null, 12, Rt)) : (a(), l("span", {
        key: 1,
        class: "w-icon__glyph",
        style: P({ fontSize: `${Math.round(t.value * 0.85)}px` })
      }, _(e.glyph), 5))
    ], 14, Nt));
  }
}), Kt = /* @__PURE__ */ W(Ot, [["__scopeId", "data-v-11d25b82"]]), Yt = {
  key: 0,
  class: "w-mnemonic"
}, Ht = {
  key: 0,
  class: "w-field__req",
  "aria-hidden": "true"
}, jt = { class: "w-field__control" }, Ut = /* @__PURE__ */ C({
  __name: "WField",
  props: {
    label: { default: "" },
    labelWidth: { default: 110 },
    required: { type: Boolean, default: !1 },
    inline: { type: Boolean, default: !1 }
  },
  setup(e) {
    const n = e, t = $(() => N(n.label)), u = $(
      () => typeof n.labelWidth == "number" ? `${n.labelWidth}px` : n.labelWidth
    );
    return (s, h) => (a(), l("div", {
      class: B(["w-field", { "is-inline": e.inline }])
    }, [
      r("div", {
        class: "w-field__label",
        style: P({ width: u.value })
      }, [
        (a(!0), l(M, null, z(t.value, (b, i) => (a(), l(M, { key: i }, [
          b.mnemonic ? (a(), l("u", Yt, _(b.ch), 1)) : (a(), l(M, { key: 1 }, [
            F(_(b.ch), 1)
          ], 64))
        ], 64))), 128)),
        e.required ? (a(), l("span", Ht, "*")) : p("", !0)
      ], 4),
      r("div", jt, [
        E(s.$slots, "default", {}, void 0, !0)
      ])
    ], 2));
  }
}), qt = /* @__PURE__ */ W(Ut, [["__scopeId", "data-v-f6e18fc7"]]), Gt = { class: "w-scrollbar__track" }, Xt = /* @__PURE__ */ C({
  __name: "WScrollBar",
  props: {
    orientation: { default: "vertical" },
    value: { default: 0 },
    min: { default: 0 },
    max: { default: 100 },
    pageSize: { default: 10 },
    disabled: { type: Boolean, default: !1 }
  },
  emits: ["update:value"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => {
      const i = t.max - t.min;
      return i <= 0 ? 0 : Math.min(1, Math.max(0, (t.value - t.min) / i));
    }), h = $(() => {
      const i = t.max - t.min;
      if (i <= 0) return 1;
      const o = t.pageSize / i;
      return Math.min(1, Math.max(0.08, o));
    });
    function b(i) {
      if (t.disabled) return;
      const o = t.max - t.min, d = Math.max(1, Math.round(o / 20)), c = Math.min(t.max, Math.max(t.min, t.value + i * d));
      c !== t.value && u("update:value", c);
    }
    return (i, o) => (a(), l("div", {
      class: B(["w-scrollbar", [`w-scrollbar--${e.orientation}`, { "is-disabled": e.disabled }]])
    }, [
      e.orientation === "vertical" ? (a(), l("button", {
        key: 0,
        type: "button",
        class: "w-scrollbar__btn w-scrollbar__btn--up",
        tabindex: "-1",
        "aria-label": "Scroll up",
        onClick: o[0] || (o[0] = (d) => b(-1))
      }, [...o[2] || (o[2] = [
        r("span", { class: "w-scrollbar__glyph w-scrollbar__glyph--up" }, null, -1)
      ])])) : p("", !0),
      r("div", Gt, [
        r("div", {
          class: "w-scrollbar__thumb",
          style: P(
            e.orientation === "vertical" ? { height: `${h.value * 100}%`, top: `${s.value * (100 - h.value * 100)}%` } : { width: `${h.value * 100}%`, left: `${s.value * (100 - h.value * 100)}%` }
          )
        }, null, 4)
      ]),
      e.orientation === "vertical" ? (a(), l("button", {
        key: 1,
        type: "button",
        class: "w-scrollbar__btn w-scrollbar__btn--down",
        tabindex: "-1",
        "aria-label": "Scroll down",
        onClick: o[1] || (o[1] = (d) => b(1))
      }, [...o[3] || (o[3] = [
        r("span", { class: "w-scrollbar__glyph w-scrollbar__glyph--down" }, null, -1)
      ])])) : p("", !0)
    ], 2));
  }
}), Jt = /* @__PURE__ */ W(Xt, [["__scopeId", "data-v-71da42c0"]]), Qt = {
  key: 0,
  class: "w-titlebar__icon",
  "aria-hidden": "true"
}, Zt = { class: "w-titlebar__text" }, ea = ["data-draggable"], ta = { class: "w-titlebar__buttons" }, aa = /* @__PURE__ */ C({
  __name: "WTitleBar",
  props: {
    title: { default: "" },
    icon: { default: "" },
    active: { type: Boolean, default: !0 },
    showMin: { type: Boolean, default: !0 },
    showMax: { type: Boolean, default: !0 },
    showClose: { type: Boolean, default: !0 },
    draggable: { type: Boolean, default: !1 },
    height: { default: 18 }
  },
  emits: ["minimize", "maximize", "close", "dblclick"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => t.active ? "is-active" : "is-inactive");
    return (h, b) => (a(), l("div", {
      class: B(["w-titlebar w-selectable", s.value]),
      style: P({ height: `${e.height}px`, "--wails-draggable": e.draggable ? "drag" : "no-drag" }),
      onDblclick: b[3] || (b[3] = (i) => u("dblclick"))
    }, [
      e.icon ? (a(), l("span", Qt, _(e.icon), 1)) : p("", !0),
      r("span", Zt, _(e.title), 1),
      r("span", {
        class: "w-titlebar__spacer",
        "data-draggable": e.draggable ? "true" : "false"
      }, null, 8, ea),
      r("span", ta, [
        e.showMin ? (a(), l("button", {
          key: 0,
          type: "button",
          class: "w-titlebar__btn",
          tabindex: "-1",
          "aria-label": "Minimize",
          onClick: b[0] || (b[0] = R((i) => u("minimize"), ["stop"]))
        }, [...b[4] || (b[4] = [
          r("span", { class: "w-titlebar__glyph w-titlebar__glyph--min" }, null, -1)
        ])])) : p("", !0),
        e.showMax ? (a(), l("button", {
          key: 1,
          type: "button",
          class: "w-titlebar__btn",
          tabindex: "-1",
          "aria-label": "Maximize",
          onClick: b[1] || (b[1] = R((i) => u("maximize"), ["stop"]))
        }, [...b[5] || (b[5] = [
          r("span", { class: "w-titlebar__glyph w-titlebar__glyph--max" }, null, -1)
        ])])) : p("", !0),
        e.showClose ? (a(), l("button", {
          key: 2,
          type: "button",
          class: "w-titlebar__btn",
          tabindex: "-1",
          "aria-label": "Close",
          onClick: b[2] || (b[2] = R((i) => u("close"), ["stop"]))
        }, [...b[6] || (b[6] = [
          r("span", { class: "w-titlebar__glyph w-titlebar__glyph--close" }, null, -1)
        ])])) : p("", !0)
      ])
    ], 38));
  }
}), la = /* @__PURE__ */ W(aa, [["__scopeId", "data-v-e216093d"]]), na = ["aria-disabled", "onClick"], ia = { class: "w-menubar__root-label" }, sa = {
  key: 0,
  class: "w-mnemonic"
}, oa = {
  key: 0,
  class: "w-menubar__popup w-bevel-raised",
  role: "menu"
}, da = {
  key: 0,
  class: "w-menubar__sep",
  role: "separator"
}, ua = ["aria-disabled", "onClick"], ca = {
  class: "w-menubar__check",
  "aria-hidden": "true"
}, ra = {
  key: 0,
  class: "w-menubar__tick"
}, ha = {
  key: 0,
  class: "w-menubar__icon"
}, fa = { class: "w-menubar__entry-label" }, ma = {
  key: 1,
  class: "w-menubar__accel"
}, ba = /* @__PURE__ */ C({
  __name: "WMenuBar",
  props: {
    model: { default: () => [] },
    height: { default: 20 }
  },
  emits: ["select"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(null), h = D(null), b = $(() => t.model);
    function i(c) {
      s.value = s.value === c ? null : c;
    }
    function o(c) {
      var y;
      (y = c.action) == null || y.call(c), u("select", c), s.value = null;
    }
    function d(c) {
      s.value !== null && h.value && !h.value.contains(c.target) && (s.value = null);
    }
    return Y(() => document.addEventListener("mousedown", d)), j(() => document.removeEventListener("mousedown", d)), (c, y) => (a(), l("div", {
      ref_key: "bar",
      ref: h,
      class: "w-menubar w-selectable",
      role: "menubar",
      style: P({ height: `${e.height}px` })
    }, [
      (a(!0), l(M, null, z(b.value, (x, w) => (a(), l("div", {
        key: w,
        class: B(["w-menubar__root", { "is-open": s.value === w, "is-disabled": x.disabled }]),
        role: "menuitem",
        "aria-disabled": x.disabled || void 0,
        onClick: (v) => !x.disabled && i(w)
      }, [
        r("span", ia, [
          (a(!0), l(M, null, z(H(N)(x.label ?? ""), (v, g) => (a(), l(M, { key: g }, [
            v.mnemonic ? (a(), l("u", sa, _(v.ch), 1)) : (a(), l(M, { key: 1 }, [
              F(_(v.ch), 1)
            ], 64))
          ], 64))), 128))
        ]),
        s.value === w ? (a(), l("div", oa, [
          (a(!0), l(M, null, z(x.children ?? [], (v, g) => (a(), l(M, { key: g }, [
            v.separator ? (a(), l("div", da)) : (a(), l("div", {
              key: 1,
              class: B(["w-menubar__entry", { "is-disabled": v.disabled, "is-checked": v.checked }]),
              role: "menuitem",
              "aria-disabled": v.disabled || void 0,
              onClick: R((f) => !v.disabled && o(v), ["stop"])
            }, [
              r("span", ca, [
                v.checked ? (a(), l("span", ra)) : p("", !0)
              ]),
              v.icon ? (a(), l("span", ha, _(v.icon), 1)) : p("", !0),
              r("span", fa, _(v.label), 1),
              v.accelerator ? (a(), l("span", ma, _(v.accelerator), 1)) : p("", !0)
            ], 10, ua))
          ], 64))), 128))
        ])) : p("", !0)
      ], 10, na))), 128))
    ], 4));
  }
}), va = /* @__PURE__ */ W(ba, [["__scopeId", "data-v-50237ff6"]]), wa = {
  class: "w-menu w-bevel-raised",
  role: "menu"
}, ya = /* @__PURE__ */ C({
  __name: "WMenu",
  props: {
    label: { default: "" }
  },
  setup(e) {
    return (n, t) => (a(), l("div", wa, [
      E(n.$slots, "default", {}, void 0, !0)
    ]));
  }
}), ga = /* @__PURE__ */ W(ya, [["__scopeId", "data-v-86b645cd"]]), _a = {
  key: 0,
  class: "w-menu__sep",
  role: "separator"
}, pa = ["aria-disabled", "aria-haspopup"], xa = {
  class: "w-menu__check",
  "aria-hidden": "true"
}, ka = {
  key: 0,
  class: "w-menu__radio-dot"
}, $a = {
  key: 1,
  class: "w-menu__tick"
}, Ma = {
  key: 0,
  class: "w-menu__icon",
  "aria-hidden": "true"
}, Ba = { class: "w-menu__label" }, Va = {
  key: 0,
  class: "w-mnemonic"
}, Wa = {
  key: 1,
  class: "w-menu__accel"
}, Ca = {
  key: 2,
  class: "w-menu__arrow",
  "aria-hidden": "true"
}, Sa = /* @__PURE__ */ C({
  __name: "WMenuItem",
  props: {
    label: { default: "" },
    accelerator: { default: "" },
    icon: { default: "" },
    checked: { type: Boolean, default: !1 },
    radio: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 },
    separator: { type: Boolean, default: !1 },
    submenu: { type: Boolean, default: !1 }
  },
  emits: ["click"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(() => N(t.label));
    return (h, b) => e.separator ? (a(), l("li", _a)) : (a(), l("li", {
      key: 1,
      class: B(["w-menu__item", { "is-disabled": e.disabled, "is-checked": e.checked }]),
      role: "menuitem",
      "aria-disabled": e.disabled || void 0,
      "aria-haspopup": e.submenu ? "menu" : void 0,
      onClick: b[0] || (b[0] = (i) => !e.disabled && u("click"))
    }, [
      r("span", xa, [
        e.radio && e.checked ? (a(), l("span", ka)) : e.checked ? (a(), l("span", $a)) : p("", !0)
      ]),
      e.icon ? (a(), l("span", Ma, _(e.icon), 1)) : p("", !0),
      r("span", Ba, [
        (a(!0), l(M, null, z(s.value, (i, o) => (a(), l(M, { key: o }, [
          i.mnemonic ? (a(), l("u", Va, _(i.ch), 1)) : (a(), l(M, { key: 1 }, [
            F(_(i.ch), 1)
          ], 64))
        ], 64))), 128))
      ]),
      e.accelerator ? (a(), l("span", Wa, _(e.accelerator), 1)) : p("", !0),
      e.submenu ? (a(), l("span", Ca, "▶")) : p("", !0)
    ], 10, pa));
  }
}), Ia = /* @__PURE__ */ W(Sa, [["__scopeId", "data-v-2e177577"]]), Pa = {
  key: 0,
  class: "w-toolbar__gripper",
  "aria-hidden": "true"
}, Ta = /* @__PURE__ */ C({
  __name: "WToolBar",
  props: {
    gripper: { type: Boolean, default: !0 },
    height: { default: 20 }
  },
  setup(e) {
    return (n, t) => (a(), l("div", {
      class: "w-toolbar w-selectable",
      style: P({ height: `${e.height}px` })
    }, [
      e.gripper ? (a(), l("span", Pa, [...t[0] || (t[0] = [
        r("i", null, null, -1),
        r("i", null, null, -1),
        r("i", null, null, -1)
      ])])) : p("", !0),
      E(n.$slots, "default", {}, void 0, !0)
    ], 4));
  }
}), za = /* @__PURE__ */ W(Ta, [["__scopeId", "data-v-d393ce44"]]), Da = ["disabled", "title"], Aa = {
  key: 0,
  class: "w-toolbtn__icon",
  "aria-hidden": "true"
}, Ea = {
  key: 1,
  class: "w-toolbtn__label"
}, La = /* @__PURE__ */ C({
  __name: "WToolButton",
  props: {
    icon: { default: "" },
    label: { default: "" },
    text: { default: "icon" },
    disabled: { type: Boolean, default: !1 },
    checked: { type: Boolean, default: !1 },
    tooltip: { default: "" }
  },
  emits: ["click"],
  setup(e, { emit: n }) {
    const t = n;
    return (u, s) => (a(), l("button", {
      type: "button",
      class: B(["w-toolbtn w-selectable", {
        "is-disabled": e.disabled,
        "is-checked": e.checked,
        "has-label": e.text !== "icon",
        "is-icon-only": e.text === "icon"
      }]),
      disabled: e.disabled,
      title: e.tooltip || void 0,
      onClick: s[0] || (s[0] = (h) => t("click", h))
    }, [
      e.icon ? (a(), l("span", Aa, _(e.icon), 1)) : p("", !0),
      e.text !== "icon" && e.label ? (a(), l("span", Ea, _(e.label), 1)) : p("", !0)
    ], 10, Da));
  }
}), Fa = /* @__PURE__ */ W(La, [["__scopeId", "data-v-b280279c"]]), Na = {}, Ra = {
  class: "w-toolsep",
  role: "separator"
};
function Oa(e, n) {
  return a(), l("span", Ra);
}
const Ka = /* @__PURE__ */ W(Na, [["render", Oa], ["__scopeId", "data-v-45c39fb6"]]), Ya = {
  key: 0,
  class: "w-statusbar__gripper",
  "aria-hidden": "true"
}, Ha = /* @__PURE__ */ C({
  __name: "WStatusBar",
  props: {
    gripper: { type: Boolean, default: !0 },
    height: { default: 20 }
  },
  setup(e) {
    return (n, t) => (a(), l("div", {
      class: "w-statusbar",
      style: P({ height: `${e.height}px` })
    }, [
      E(n.$slots, "default", {}, void 0, !0),
      e.gripper ? (a(), l("span", Ya, [...t[0] || (t[0] = [
        r("i", null, null, -1),
        r("i", null, null, -1),
        r("i", null, null, -1)
      ])])) : p("", !0)
    ], 4));
  }
}), ja = /* @__PURE__ */ W(Ha, [["__scopeId", "data-v-fac7a8eb"]]), Ua = /* @__PURE__ */ C({
  __name: "WStatusPane",
  props: {
    width: { default: "auto" },
    align: { default: "left" }
  },
  setup(e) {
    return (n, t) => (a(), l("div", {
      class: B(["w-statuspane", `is-${e.align}`]),
      style: P({
        width: typeof e.width == "number" ? e.width + "px" : e.width,
        flex: e.width === "auto" ? "1 1 auto" : "0 0 auto"
      })
    }, [
      E(n.$slots, "default", {}, void 0, !0)
    ], 6));
  }
}), qa = /* @__PURE__ */ W(Ua, [["__scopeId", "data-v-38f83d6f"]]), Ga = ["aria-selected", "onClick"], Xa = {
  key: 0,
  class: "w-tab__icon",
  "aria-hidden": "true"
}, Ja = { class: "w-tab__label" }, Qa = /* @__PURE__ */ C({
  __name: "WTabBar",
  props: {
    model: { default: () => [] },
    modelValue: { default: "" },
    position: { default: "top" }
  },
  emits: ["update:modelValue", "change"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(t.modelValue);
    L(
      () => t.modelValue,
      (i) => {
        i && (s.value = i);
      },
      { immediate: !0 }
    ), L(
      () => t.model,
      (i) => {
        i.length && (i.some((o) => o.id === s.value) || (s.value = i[0].id));
      },
      { immediate: !0 }
    );
    const h = $(() => s.value);
    function b(i) {
      i.disabled || (s.value = i.id, u("update:modelValue", i.id), u("change", i.id));
    }
    return (i, o) => (a(), l("div", {
      class: B(["w-tabbar", `w-tabbar--${e.position}`]),
      role: "tablist"
    }, [
      (a(!0), l(M, null, z(e.model, (d) => (a(), l("div", {
        key: d.id,
        class: B(["w-tab", { "is-active": h.value === d.id, "is-disabled": d.disabled }]),
        role: "tab",
        "aria-selected": h.value === d.id,
        onClick: (c) => b(d)
      }, [
        d.icon ? (a(), l("span", Xa, _(d.icon), 1)) : p("", !0),
        r("span", Ja, _(d.label), 1)
      ], 10, Ga))), 128))
    ], 2));
  }
}), Za = /* @__PURE__ */ W(Qa, [["__scopeId", "data-v-7336e532"]]), el = ["aria-selected"], tl = {
  key: 0,
  class: "w-tab__icon",
  "aria-hidden": "true"
}, al = { class: "w-tab__label" }, ll = /* @__PURE__ */ C({
  __name: "WTab",
  props: {
    label: { default: "" },
    icon: { default: "" },
    active: { type: Boolean, default: !1 },
    disabled: { type: Boolean, default: !1 }
  },
  emits: ["click"],
  setup(e, { emit: n }) {
    const t = n;
    return (u, s) => (a(), l("div", {
      class: B(["w-tab", { "is-active": e.active, "is-disabled": e.disabled }]),
      role: "tab",
      "aria-selected": e.active,
      onClick: s[0] || (s[0] = (h) => !e.disabled && t("click"))
    }, [
      e.icon ? (a(), l("span", tl, _(e.icon), 1)) : p("", !0),
      r("span", al, _(e.label), 1)
    ], 10, el));
  }
}), nl = /* @__PURE__ */ W(ll, [["__scopeId", "data-v-96fbc6c2"]]), il = {
  key: 0,
  class: "w-dialog__icon",
  "aria-hidden": "true"
}, sl = { class: "w-dialog__title-text" }, ol = { class: "w-dialog__body" }, dl = {
  key: 0,
  class: "w-dialog__footer"
}, ul = /* @__PURE__ */ C({
  __name: "WDialog",
  props: {
    title: { default: "" },
    icon: { default: "" },
    width: { default: "360px" },
    modal: { type: Boolean, default: !1 },
    closable: { type: Boolean, default: !0 },
    active: { type: Boolean, default: !0 }
  },
  emits: ["close"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(
      () => typeof t.width == "number" ? `${t.width}px` : t.width
    );
    return (h, b) => (a(), l("div", {
      class: B(["w-dialog-layer", { "is-modal": e.modal }])
    }, [
      r("div", {
        class: "w-dialog w-bevel-window",
        style: P({ width: s.value }),
        role: "dialog"
      }, [
        r("div", {
          class: B(["w-dialog__title", { "is-active": e.active }])
        }, [
          e.icon ? (a(), l("span", il, _(e.icon), 1)) : p("", !0),
          r("span", sl, _(e.title), 1),
          e.closable ? (a(), l("button", {
            key: 1,
            type: "button",
            class: "w-dialog__close",
            tabindex: "-1",
            "aria-label": "Close",
            onClick: b[0] || (b[0] = (i) => u("close"))
          }, [...b[1] || (b[1] = [
            r("span", { class: "w-dialog__close-glyph" }, null, -1)
          ])])) : p("", !0)
        ], 2),
        r("div", ol, [
          E(h.$slots, "default", {}, void 0, !0)
        ]),
        h.$slots.footer ? (a(), l("div", dl, [
          E(h.$slots, "footer", {}, void 0, !0)
        ])) : p("", !0)
      ], 4)
    ], 2));
  }
}), cl = /* @__PURE__ */ W(ul, [["__scopeId", "data-v-6f5683d1"]]), rl = { class: "w-msgbox__title" }, hl = { class: "w-msgbox__title-text" }, fl = { class: "w-msgbox__body" }, ml = { class: "w-msgbox__glyph" }, bl = { class: "w-msgbox__message" }, vl = { class: "w-msgbox__footer" }, wl = ["onClick"], yl = /* @__PURE__ */ C({
  __name: "WMessageBox",
  props: {
    type: { default: "info" },
    title: { default: "Message" },
    message: { default: "" },
    buttons: { default: () => ["OK"] },
    width: { default: "320px" }
  },
  emits: ["action"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = $(
      () => typeof t.width == "number" ? `${t.width}px` : t.width
    );
    return (h, b) => (a(), l("div", {
      class: "w-msgbox w-bevel-window",
      style: P({ width: s.value }),
      role: "alertdialog"
    }, [
      r("div", rl, [
        r("span", hl, _(e.title), 1)
      ]),
      r("div", fl, [
        r("span", {
          class: B(["w-msgbox__icon", `is-${e.type}`]),
          "aria-hidden": "true"
        }, [
          r("span", ml, _(e.type === "error" ? "×" : e.type === "warning" ? "!" : e.type === "question" ? "?" : "i"), 1)
        ], 2),
        r("p", bl, _(e.message), 1)
      ]),
      r("div", vl, [
        (a(!0), l(M, null, z(e.buttons, (i) => (a(), l("button", {
          key: i,
          type: "button",
          class: B(["w-msgbox__btn", { "is-default": i === e.buttons[0] }]),
          onClick: (o) => u("action", i)
        }, _(i), 11, wl))), 128))
      ])
    ], 4));
  }
}), gl = /* @__PURE__ */ W(yl, [["__scopeId", "data-v-72630c1a"]]), _l = { class: "w-groupbox" }, pl = { class: "w-groupbox__legend" }, xl = { class: "w-groupbox__content" }, kl = /* @__PURE__ */ C({
  __name: "WGroupBox",
  props: {
    label: { default: "" }
  },
  setup(e) {
    return (n, t) => (a(), l("fieldset", _l, [
      r("legend", pl, _(e.label), 1),
      r("div", xl, [
        E(n.$slots, "default", {}, void 0, !0)
      ])
    ]));
  }
}), $l = /* @__PURE__ */ W(kl, [["__scopeId", "data-v-b395c86e"]]), Ml = ["aria-orientation", "aria-valuenow"], Bl = /* @__PURE__ */ C({
  __name: "WSplitter",
  props: {
    orientation: { default: "vertical" },
    size: { default: 4 },
    modelValue: { default: 50 },
    min: { default: 0 },
    max: { default: 100 },
    disabled: { type: Boolean, default: !1 }
  },
  emits: ["update:modelValue"],
  setup(e, { emit: n }) {
    const t = e, u = n;
    $(() => {
      const h = t.max - t.min;
      return h <= 0 ? 50 : (t.modelValue - t.min) / h * 100;
    });
    function s(h) {
      if (t.disabled) return;
      const i = h.currentTarget.getBoundingClientRect(), o = t.orientation === "horizontal" ? (h.clientX - i.left) / Math.max(1, i.width) : (h.clientY - i.top) / Math.max(1, i.height), d = Math.round((t.min + Math.min(1, Math.max(0, o)) * (t.max - t.min)) * 100) / 100;
      u("update:modelValue", d);
    }
    return (h, b) => (a(), l("div", {
      class: B(["w-splitter", [`w-splitter--${e.orientation}`, { "is-disabled": e.disabled }]]),
      role: "separator",
      "aria-orientation": e.orientation,
      "aria-valuenow": e.modelValue,
      tabindex: "0",
      onPointerdown: s
    }, [...b[0] || (b[0] = [
      r("span", {
        class: "w-splitter__grip",
        "aria-hidden": "true"
      }, [
        r("i"),
        r("i"),
        r("i")
      ], -1)
    ])], 42, Ml));
  }
}), Vl = /* @__PURE__ */ W(Bl, [["__scopeId", "data-v-e2b87b3a"]]), Wl = {
  key: 0,
  class: "w-panel__title"
}, Cl = { class: "w-panel__body" }, Sl = /* @__PURE__ */ C({
  __name: "WPanel",
  props: {
    variant: { default: "face" },
    padding: { default: 6 },
    title: { default: "" }
  },
  setup(e) {
    const n = e, t = $(
      () => typeof n.padding == "number" ? `${n.padding}px` : n.padding
    );
    return (u, s) => (a(), l("section", {
      class: B(["w-panel", `w-panel--${e.variant}`]),
      style: P({ padding: t.value })
    }, [
      e.title ? (a(), l("header", Wl, _(e.title), 1)) : p("", !0),
      r("div", Cl, [
        E(u.$slots, "default", {}, void 0, !0)
      ])
    ], 6));
  }
}), Il = /* @__PURE__ */ W(Sl, [["__scopeId", "data-v-9501efa4"]]), Pl = {
  key: 0,
  class: "w-lamp__label"
}, Tl = /* @__PURE__ */ C({
  __name: "WStatusLamp",
  props: {
    state: { default: "off" },
    color: { default: "green" },
    label: { default: "" },
    size: { default: 16 }
  },
  setup(e) {
    const n = e, t = $(() => n.state === "on" || n.state === "blink"), u = $(() => n.state === "fault" ? "is-fault" : n.state === "blink" ? "is-blink" : t.value ? "is-on" : "is-off");
    return (s, h) => (a(), l("span", {
      class: B(["w-lamp", u.value])
    }, [
      r("span", {
        class: B(["w-lamp__bulb", `is-${e.color}`]),
        style: P({ width: `${e.size}px`, height: `${e.size}px` })
      }, null, 6),
      e.label ? (a(), l("span", Pl, _(e.label), 1)) : p("", !0)
    ], 2));
  }
}), zl = /* @__PURE__ */ W(Tl, [["__scopeId", "data-v-b2f85672"]]);
function Dl(e, n) {
  return Number.isFinite(e) ? n <= 5 ? e.toFixed(2) : n <= 50 ? e.toFixed(1) : e.toFixed(0) : "--";
}
const Al = { class: "w-gauge" }, El = /* @__PURE__ */ C({
  __name: "WGauge",
  props: {
    value: { default: 0 },
    min: { default: 0 },
    max: { default: 100 },
    unit: { default: "" },
    label: { default: "" },
    zones: { default: () => [] },
    size: { default: 120 }
  },
  setup(e) {
    const n = e, t = D(null), u = $(() => {
      const i = n.max - n.min;
      return i <= 0 ? 0 : Math.min(1, Math.max(0, (n.value - n.min) / i));
    }), s = Math.PI * 0.75, h = Math.PI * 1.5;
    function b() {
      const i = t.value;
      if (!i) return;
      const o = window.devicePixelRatio || 1, d = n.size;
      i.width = d * o, i.height = d * o, i.style.width = `${d}px`, i.style.height = `${d}px`;
      const c = i.getContext("2d");
      if (!c) return;
      c.setTransform(o, 0, 0, o, 0, 0), c.clearRect(0, 0, d, d);
      const y = d / 2, x = d / 2, w = d / 2 - 8;
      c.fillStyle = "#ffffff", c.beginPath(), c.arc(y, x, w + 5, 0, Math.PI * 2), c.fill(), c.strokeStyle = "#808080", c.lineWidth = 2, c.beginPath(), c.arc(y, x, w + 5, 0, Math.PI * 2), c.stroke(), c.strokeStyle = "#ffffff", c.lineWidth = 1, c.beginPath(), c.arc(y, x, w + 3.5, 0, Math.PI * 2), c.stroke();
      for (const g of n.zones) {
        const f = (g.from - n.min) / (n.max - n.min), m = (g.to - n.min) / (n.max - n.min);
        c.strokeStyle = g.color, c.lineWidth = 4, c.beginPath(), c.arc(y, x, w - 1, s + f * h, s + m * h), c.stroke();
      }
      c.strokeStyle = "#000000";
      for (let g = 0; g <= 10; g++) {
        const f = s + g / 10 * h, m = g % 2 === 0, V = w - (m ? 12 : 6);
        c.lineWidth = m ? 1.5 : 1, c.beginPath(), c.moveTo(y + Math.cos(f) * V, x + Math.sin(f) * V), c.lineTo(y + Math.cos(f) * (w - 4), x + Math.sin(f) * (w - 4)), c.stroke();
      }
      const v = s + u.value * h;
      c.strokeStyle = "#000000", c.lineWidth = 2, c.beginPath(), c.moveTo(y - Math.cos(v) * 6, x - Math.sin(v) * 6), c.lineTo(y + Math.cos(v) * (w - 8), x + Math.sin(v) * (w - 8)), c.stroke(), c.fillStyle = "#000000", c.beginPath(), c.arc(y, x, 3, 0, Math.PI * 2), c.fill(), c.fillStyle = "#000000", c.font = "bold 11px Tahoma, sans-serif", c.textAlign = "center", c.fillText(Dl(n.value, n.max - n.min) + n.unit, y, x + w * 0.55), n.label && (c.font = "10px Tahoma, sans-serif", c.fillText(n.label, y, x + w * 0.55 + 13));
    }
    return Y(b), L(() => [n.value, n.min, n.max, n.size, n.label, n.unit], b), L(() => n.zones, b, { deep: !0 }), (i, o) => (a(), l("div", Al, [
      r("canvas", {
        ref_key: "canvas",
        ref: t,
        class: "w-gauge__canvas"
      }, null, 512)
    ]));
  }
}), Ll = /* @__PURE__ */ W(El, [["__scopeId", "data-v-f1e97c3a"]]), Fl = { class: "w-knob" }, Nl = ["aria-valuemin", "aria-valuemax", "aria-valuenow", "aria-label"], Rl = { class: "w-knob__caption" }, Ol = { class: "w-knob__value w-mono" }, Kl = {
  key: 0,
  class: "w-knob__label"
}, Yl = /* @__PURE__ */ C({
  __name: "WKnob",
  props: {
    modelValue: { default: 0 },
    min: { default: 0 },
    max: { default: 100 },
    label: { default: "" },
    unit: { default: "" },
    size: { default: 64 },
    startAngle: { default: -135 },
    endAngle: { default: 135 },
    disabled: { type: Boolean, default: !1 }
  },
  emits: ["update:modelValue"],
  setup(e, { emit: n }) {
    const t = e, u = n, s = D(null), h = D(!1), b = $(() => {
      const v = t.max - t.min;
      return v <= 0 ? 0 : Math.min(1, Math.max(0, (t.modelValue - t.min) / v));
    });
    function i(v) {
      return t.startAngle + (t.endAngle - t.startAngle) * v;
    }
    function o() {
      const v = s.value;
      if (!v) return;
      const g = window.devicePixelRatio || 1, f = t.size;
      v.width = f * g, v.height = f * g, v.style.width = `${f}px`, v.style.height = `${f}px`;
      const m = v.getContext("2d");
      if (!m) return;
      m.setTransform(g, 0, 0, g, 0, 0), m.clearRect(0, 0, f, f);
      const V = f / 2, T = f / 2, k = f / 2 - 3;
      m.fillStyle = "#d4d0c8", m.beginPath(), m.arc(V, T, k, 0, Math.PI * 2), m.fill(), m.strokeStyle = "#808080", m.lineWidth = 2, m.stroke(), m.strokeStyle = "#ffffff", m.lineWidth = 1, m.beginPath(), m.arc(V, T, k - 1, 0, Math.PI * 2), m.stroke();
      for (let I = 0; I <= 20; I++) {
        const A = i(I / 20), O = I % 5 === 0, K = k - (O ? 9 : 5);
        m.strokeStyle = "#000000", m.lineWidth = O ? 1.5 : 1, m.beginPath(), m.moveTo(V + Math.cos(A * Math.PI / 180) * K, T + Math.sin(A * Math.PI / 180) * K), m.lineTo(V + Math.cos(A * Math.PI / 180) * (k - 3), T + Math.sin(A * Math.PI / 180) * (k - 3)), m.stroke();
      }
      const S = i(b.value) * Math.PI / 180;
      m.strokeStyle = "#000000", m.lineWidth = 2, m.beginPath(), m.moveTo(V + Math.cos(S) * 3, T + Math.sin(S) * 3), m.lineTo(V + Math.cos(S) * (k - 10), T + Math.sin(S) * (k - 10)), m.stroke(), m.strokeStyle = "#808080", m.lineWidth = 1;
      for (let I = 0; I < 24; I++) {
        const A = I / 24 * Math.PI * 2;
        m.beginPath(), m.moveTo(V + Math.cos(A) * (k - 3), T + Math.sin(A) * (k - 3)), m.lineTo(V + Math.cos(A) * k, T + Math.sin(A) * k), m.stroke();
      }
    }
    function d(v) {
      const g = s.value;
      if (!g) return t.modelValue;
      const f = g.getBoundingClientRect(), m = v.clientX - (f.left + f.width / 2), V = v.clientY - (f.top + f.height / 2);
      let T = Math.atan2(V, m) * 180 / Math.PI + 90;
      T < t.startAngle && (T = t.startAngle), T > t.endAngle && (T = t.endAngle);
      const k = (T - t.startAngle) / (t.endAngle - t.startAngle), S = t.min + k * (t.max - t.min);
      return Math.round(S * 100) / 100;
    }
    function c(v) {
      var g, f;
      t.disabled || (h.value = !0, (f = (g = v.currentTarget).setPointerCapture) == null || f.call(g, v.pointerId), u("update:modelValue", d(v)));
    }
    function y(v) {
      !h.value || t.disabled || u("update:modelValue", d(v));
    }
    function x(v) {
      var g, f;
      h.value && (h.value = !1, (f = (g = v.currentTarget).releasePointerCapture) == null || f.call(g, v.pointerId));
    }
    function w(v) {
      if (t.disabled) return;
      const g = t.max - t.min;
      let f = null;
      (v.key === "ArrowUp" || v.key === "ArrowRight") && (f = t.modelValue + g / 50), (v.key === "ArrowDown" || v.key === "ArrowLeft") && (f = t.modelValue - g / 50), f !== null && (v.preventDefault(), u("update:modelValue", Math.min(t.max, Math.max(t.min, f))));
    }
    return Y(o), L(() => [t.modelValue, t.min, t.max, t.size], o), (v, g) => (a(), l("div", Fl, [
      r("canvas", {
        ref_key: "canvas",
        ref: s,
        class: "w-knob__canvas",
        tabindex: "0",
        role: "slider",
        "aria-valuemin": e.min,
        "aria-valuemax": e.max,
        "aria-valuenow": e.modelValue,
        "aria-label": e.label || void 0,
        onPointerdown: c,
        onPointermove: y,
        onPointerup: x,
        onPointercancel: x,
        onKeydown: w
      }, null, 40, Nl),
      r("div", Rl, [
        r("span", Ol, _(e.modelValue) + _(e.unit), 1),
        e.label ? (a(), l("span", Kl, _(e.label), 1)) : p("", !0)
      ])
    ]));
  }
}), Hl = /* @__PURE__ */ W(Yl, [["__scopeId", "data-v-c08c8878"]]), jl = {
  class: "w-segdisplay__svg",
  viewBox: "0 0 10 18",
  preserveAspectRatio: "none",
  "shape-rendering": "crispEdges"
}, Ul = { class: "w-segdisplay__on" }, ql = /* @__PURE__ */ C({
  __name: "WSegmentDisplay",
  props: {
    value: { default: "" },
    digits: { default: 4 },
    pad: { type: Boolean, default: !0 },
    color: { default: "red" },
    size: { default: 16 },
    showSign: { type: Boolean, default: !1 }
  },
  setup(e) {
    const n = e, t = {
      0: ["a", "b", "c", "d", "e", "f"],
      1: ["b", "c"],
      2: ["a", "b", "g", "e", "d"],
      3: ["a", "b", "g", "c", "d"],
      4: ["f", "g", "b", "c"],
      5: ["a", "f", "g", "c", "d"],
      6: ["a", "f", "g", "e", "c", "d"],
      7: ["a", "b", "c"],
      8: ["a", "b", "c", "d", "e", "f", "g"],
      9: ["a", "b", "c", "d", "f", "g"],
      "-": ["g"],
      " ": []
    }, u = ["a", "b", "c", "d", "e", "f", "g"], s = {
      a: { x: 1, y: 0, width: 8, height: 2 },
      b: { x: 8, y: 1, width: 2, height: 7 },
      c: { x: 8, y: 10, width: 2, height: 7 },
      d: { x: 1, y: 16, width: 8, height: 2 },
      e: { x: 0, y: 10, width: 2, height: 7 },
      f: { x: 0, y: 1, width: 2, height: 7 },
      g: { x: 1, y: 8, width: 8, height: 2 }
    }, h = $(() => {
      let o = String(n.value ?? "");
      if (n.showSign && !/^-/.test(o) && (o = `-${o}`), n.pad) {
        const d = o.startsWith("-"), c = d ? o.slice(1) : o;
        o = (d ? "-" : "") + c.padStart(n.digits, " ");
      }
      return o.split("");
    });
    function b(o, d) {
      const c = t[o];
      return c ? c.includes(d) : !1;
    }
    function i(o) {
      return s[o] ?? {};
    }
    return (o, d) => (a(), l("div", {
      class: B(["w-segdisplay", `is-${e.color}`]),
      style: P({ "--seg-size": `${e.size}px` })
    }, [
      (a(!0), l(M, null, z(h.value, (c, y) => (a(), l("div", {
        key: y,
        class: "w-segdisplay__digit"
      }, [
        (a(), l("svg", jl, [
          d[0] || (d[0] = X('<g class="w-segdisplay__off" data-v-4e1f491f><rect x="1" y="0" width="8" height="2" data-v-4e1f491f></rect><rect x="0" y="1" width="2" height="7" data-v-4e1f491f></rect><rect x="8" y="1" width="2" height="7" data-v-4e1f491f></rect><rect x="1" y="8" width="8" height="2" data-v-4e1f491f></rect><rect x="0" y="10" width="2" height="7" data-v-4e1f491f></rect><rect x="8" y="10" width="2" height="7" data-v-4e1f491f></rect><rect x="1" y="16" width="8" height="2" data-v-4e1f491f></rect></g>', 1)),
          r("g", Ul, [
            (a(), l(M, null, z(u, (x) => (a(), l(M, { key: x }, [
              b(c, x) ? (a(), l("rect", U({
                key: 0,
                ref_for: !0
              }, i(x)), null, 16)) : p("", !0)
            ], 64))), 64))
          ])
        ]))
      ]))), 128))
    ], 6));
  }
}), Gl = /* @__PURE__ */ W(ql, [["__scopeId", "data-v-4e1f491f"]]), Xl = { class: "w-trend" }, Jl = { class: "w-trend__canvas-wrap" }, Ql = ["viewBox", "width", "height"], Zl = ["width", "height"], en = {
  key: 0,
  class: "w-trend__grid"
}, tn = ["y1", "x2", "y2"], an = ["x1", "x2", "y2"], ln = ["points", "fill"], nn = ["points", "stroke"], sn = ["width", "height"], on = {
  key: 0,
  class: "w-trend__legend"
}, dn = /* @__PURE__ */ C({
  __name: "WTrendChart",
  props: {
    series: { default: () => [] },
    min: { default: 0 },
    max: { default: 100 },
    height: { default: 140 },
    width: { default: 320 },
    grid: { type: Boolean, default: !0 },
    showLegend: { type: Boolean, default: !0 },
    fill: { type: Boolean, default: !1 }
  },
  setup(e) {
    const n = e, t = $(() => n.max - n.min || 1);
    function u(i, o) {
      return o === 1 ? 0 : n.width / (o - 1) * i;
    }
    function s(i) {
      return n.height - (i - n.min) / t.value * n.height;
    }
    function h(i) {
      const o = i.points.length;
      return i.points.map((d, c) => `${u(c, o).toFixed(1)},${s(d).toFixed(1)}`).join(" ");
    }
    function b(i) {
      const o = i.points.length, d = i.points.map((c, y) => `${u(y, o).toFixed(1)},${s(c).toFixed(1)}`).join(" ");
      return `0,${n.height} ${d} ${n.width},${n.height}`;
    }
    return (i, o) => (a(), l("div", Xl, [
      r("div", Jl, [
        (a(), l("svg", {
          class: "w-trend__svg",
          viewBox: `0 0 ${e.width} ${e.height}`,
          width: e.width,
          height: e.height,
          preserveAspectRatio: "none"
        }, [
          r("rect", {
            x: "0",
            y: "0",
            width: e.width,
            height: e.height,
            fill: "#ffffff"
          }, null, 8, Zl),
          e.grid ? (a(), l("g", en, [
            (a(), l(M, null, z(5, (d) => r("line", {
              key: `h${d}`,
              x1: "0",
              y1: e.height / 5 * (d - 1),
              x2: e.width,
              y2: e.height / 5 * (d - 1)
            }, null, 8, tn)), 64)),
            (a(), l(M, null, z(6, (d) => r("line", {
              key: `v${d}`,
              x1: e.width / 6 * (d - 1),
              y1: "0",
              x2: e.width / 6 * (d - 1),
              y2: e.height
            }, null, 8, an)), 64))
          ])) : p("", !0),
          (a(!0), l(M, null, z(e.series, (d, c) => (a(), l("g", { key: c }, [
            d.points.length ? (a(), l(M, { key: 0 }, [
              e.fill ? (a(), l("polygon", {
                key: 0,
                points: b(d),
                fill: d.color,
                "fill-opacity": "0.15"
              }, null, 8, ln)) : p("", !0),
              r("polyline", {
                points: h(d),
                fill: "none",
                stroke: d.color,
                "stroke-width": "1.5"
              }, null, 8, nn)
            ], 64)) : p("", !0)
          ]))), 128)),
          r("rect", {
            x: "0",
            y: "0",
            width: e.width,
            height: e.height,
            fill: "none",
            stroke: "#808080",
            "stroke-width": "1"
          }, null, 8, sn)
        ], 8, Ql))
      ]),
      e.showLegend ? (a(), l("div", on, [
        (a(!0), l(M, null, z(e.series, (d, c) => (a(), l("span", {
          key: c,
          class: "w-trend__legend-item"
        }, [
          r("i", {
            class: "w-trend__swatch",
            style: P({ background: d.color })
          }, null, 4),
          F(" " + _(d.name), 1)
        ]))), 128))
      ])) : p("", !0)
    ]));
  }
}), un = /* @__PURE__ */ W(dn, [["__scopeId", "data-v-43ab4194"]]), cn = ["width", "height"], rn = ["y", "height"], hn = ["y", "height"], fn = {
  key: 0,
  class: "w-valve__label"
}, mn = { class: "w-valve__state" }, bn = /* @__PURE__ */ C({
  __name: "WValve",
  props: {
    open: { type: Boolean, default: !1 },
    label: { default: "" },
    size: { default: 48 }
  },
  setup(e) {
    return (n, t) => (a(), l("span", {
      class: B(["w-valve", { "is-open": e.open }])
    }, [
      (a(), l("svg", {
        class: "w-valve__svg",
        width: e.size,
        height: e.size * 0.8,
        viewBox: "0 0 60 48",
        "shape-rendering": "crispEdges"
      }, [
        t[0] || (t[0] = r("rect", {
          x: "0",
          y: "16",
          width: "5",
          height: "14",
          fill: "#d4d0c8",
          stroke: "#000000"
        }, null, -1)),
        t[1] || (t[1] = r("rect", {
          x: "55",
          y: "16",
          width: "5",
          height: "14",
          fill: "#d4d0c8",
          stroke: "#000000"
        }, null, -1)),
        t[2] || (t[2] = r("rect", {
          x: "4",
          y: "20",
          width: "52",
          height: "6",
          fill: "#d4d0c8",
          stroke: "#000000"
        }, null, -1)),
        t[3] || (t[3] = r("rect", {
          x: "24",
          y: "14",
          width: "12",
          height: "6",
          fill: "#d4d0c8",
          stroke: "#000000"
        }, null, -1)),
        r("rect", {
          x: "26",
          y: e.open ? 8 : 20,
          width: "8",
          height: e.open ? 6 : 12,
          fill: "#d4d0c8",
          stroke: "#000000"
        }, null, 8, rn),
        r("rect", {
          x: "28",
          y: e.open ? 2 : 20,
          width: "4",
          height: e.open ? 6 : 0,
          fill: "#808080"
        }, null, 8, hn),
        t[4] || (t[4] = r("rect", {
          x: "21",
          y: "0",
          width: "18",
          height: "4",
          fill: "#d4d0c8",
          stroke: "#000000"
        }, null, -1)),
        t[5] || (t[5] = r("rect", {
          x: "28",
          y: "4",
          width: "4",
          height: "4",
          fill: "#d4d0c8",
          stroke: "#000000"
        }, null, -1))
      ], 8, cn)),
      e.label ? (a(), l("span", fn, _(e.label), 1)) : p("", !0),
      r("span", mn, _(e.open ? "OPEN" : "SHUT"), 1)
    ], 2));
  }
}), vn = /* @__PURE__ */ W(bn, [["__scopeId", "data-v-e2e66bf9"]]), wn = {
  WButton: le,
  WCheckBox: re,
  WRadioButton: ye,
  WEdit: pe,
  WTextArea: $e,
  WComboBox: Pe,
  WListBox: Ee,
  WUpDown: Ye,
  WTrackbar: qe,
  WDateTime: nt,
  WSearchBox: dt,
  WListView: vt,
  WTreeView: Mt,
  WProgressBar: Wt,
  WLabel: Pt,
  WLink: Dt,
  WSeparator: Ft,
  WIcon: Kt,
  WField: qt,
  WScrollBar: Jt,
  WTitleBar: la,
  WMenuBar: va,
  WMenu: ga,
  WMenuItem: Ia,
  WToolBar: za,
  WToolButton: Fa,
  WToolSeparator: Ka,
  WStatusBar: ja,
  WStatusPane: qa,
  WTabBar: Za,
  WTab: nl,
  WDialog: cl,
  WMessageBox: gl,
  WGroupBox: $l,
  WSplitter: Vl,
  WPanel: Il,
  WStatusLamp: zl,
  WGauge: Ll,
  WKnob: Hl,
  WSegmentDisplay: Gl,
  WTrendChart: un,
  WValve: vn
}, gn = {
  install(e) {
    for (const [n, t] of Object.entries(wn))
      e.component(n, t);
  }
};
export {
  le as WButton,
  re as WCheckBox,
  Pe as WComboBox,
  nt as WDateTime,
  cl as WDialog,
  pe as WEdit,
  qt as WField,
  Ll as WGauge,
  $l as WGroupBox,
  Kt as WIcon,
  Hl as WKnob,
  Pt as WLabel,
  Dt as WLink,
  Ee as WListBox,
  vt as WListView,
  ga as WMenu,
  va as WMenuBar,
  Ia as WMenuItem,
  gl as WMessageBox,
  Il as WPanel,
  Wt as WProgressBar,
  ye as WRadioButton,
  Jt as WScrollBar,
  dt as WSearchBox,
  Gl as WSegmentDisplay,
  Ft as WSeparator,
  Vl as WSplitter,
  ja as WStatusBar,
  zl as WStatusLamp,
  qa as WStatusPane,
  nl as WTab,
  Za as WTabBar,
  $e as WTextArea,
  la as WTitleBar,
  za as WToolBar,
  Fa as WToolButton,
  Ka as WToolSeparator,
  qe as WTrackbar,
  Mt as WTreeView,
  un as WTrendChart,
  Ye as WUpDown,
  vn as WValve,
  gn as Win32,
  gn as default,
  N as splitMnemonic,
  J as stripMnemonic
};
